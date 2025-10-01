package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	configDir  = ".config/shelly"
	configFile = "config.json"
	apiURL     = "https://openrouter.ai/api/v1/chat/completions"
	model      = "deepseek/deepseek-chat-v3.1:free"
)

type Config struct {
	APIKey string `json:"api_key"`
}

type APIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type APIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func main() {
	debugFlag := flag.String("d", "", "Debug a command (alias for --debug)")
	debugLongFlag := flag.String("debug", "", "Debug a command")
	commandFlag := flag.String("c", "", "Generate command suggestions (alias for --command)")
	commandLongFlag := flag.String("command", "", "Generate command suggestions")
	questionFlag := flag.String("q", "", "Ask a question (alias for --question)")
	questionLongFlag := flag.String("question", "", "Ask a question")
	initFlag := flag.Bool("init", false, "Initialize shelly with API key")

	flag.Parse()

	if *initFlag {
		initializeConfig()
		return
	}

	debug := getFirstNonEmpty(*debugFlag, *debugLongFlag)
	command := getFirstNonEmpty(*commandFlag, *commandLongFlag)
	question := getFirstNonEmpty(*questionFlag, *questionLongFlag)

	modesSet := 0
	if debug != "" {
		modesSet++
	}
	if command != "" {
		modesSet++
	}
	if question != "" {
		modesSet++
	}

	if modesSet == 0 {
		printUsage()
		os.Exit(1)
	}

	if modesSet > 1 {
		fmt.Fprintln(os.Stderr, "Error: Only one mode can be used at a time")
		os.Exit(1)
	}

	config, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run 'shelly --init' to set up your API key")
		os.Exit(1)
	}

	if debug != "" {
		handleDebugMode(config, debug)
	} else if command != "" {
		handleCommandMode(config, command)
	} else if question != "" {
		handleQuestionMode(config, question)
	}
}

func getFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func printUsage() {
	fmt.Println("Shelly - Your terminal command assistant")
	fmt.Println("\nUsage:")
	fmt.Println("  shelly --init                              Initialize with API key")
	fmt.Println("  <command> | shelly -d \"description\"        Debug a command")
	fmt.Println("  shelly -c \"what you want to do\"            Generate command suggestions")
	fmt.Println("  shelly -q \"your question\"                  Ask a question")
	fmt.Println("\nFlags:")
	fmt.Println("  -d, --debug     Debug mode: analyze and fix a command")
	fmt.Println("  -c, --command   Command mode: generate command suggestions")
	fmt.Println("  -q, --question  Question mode: answer a question")
	fmt.Println("  --init          Initialize shelly with your API key")
}

func initializeConfig() {
	fmt.Println("Initializing Shelly...")
	fmt.Print("Enter your OpenRouter API key: ")

	reader := bufio.NewReader(os.Stdin)
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: API key cannot be empty")
		os.Exit(1)
	}

	config := Config{APIKey: apiKey}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	configPath := filepath.Join(homeDir, configDir)
	if err := os.MkdirAll(configPath, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		os.Exit(1)
	}

	configFilePath := filepath.Join(configPath, configFile)
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling config: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(configFilePath, data, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration saved to %s\n", configFilePath)
	fmt.Println("Shelly is ready to use!")
}

func loadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	configFilePath := filepath.Join(homeDir, configDir, configFile)
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return &config, nil
}

func handleDebugMode(config *Config, description string) {
	stdinInfo, _ := os.Stdin.Stat()
	var inputCommand string

	if (stdinInfo.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		inputCommand = strings.Join(lines, "\n")
	}

	prompt := fmt.Sprintf(`Command: %s
Context: %s

In one short sentence, explain what's wrong. Then provide exactly 3 alternative commands numbered 1-3. Format:

[one short sentence explanation]

1. [command]
2. [command]
3. [command]

Be concise. Order from best to worst. No backticks or markdown formatting.`, inputCommand, description)

	response := callAPI(config, prompt)
	fmt.Println(response)

	selectAndCopyCommand(response)
}

func handleCommandMode(config *Config, request string) {
	prompt := fmt.Sprintf(`Generate 3 terminal commands for: %s

ONLY output numbered commands, no other text:

1. [command]
2. [command]
3. [command]

Order from best to worst. No backticks or markdown.`, request)

	response := callAPI(config, prompt)
	fmt.Println(response)

	selectAndCopyCommand(response)
}

func handleQuestionMode(config *Config, question string) {
	stdinInfo, _ := os.Stdin.Stat()
	var inputContext string

	if (stdinInfo.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		inputContext = strings.Join(lines, "\n")
	}

	var prompt string
	if inputContext != "" {
		prompt = fmt.Sprintf(`Context: %s

Question: %s

Answer briefly based on the context. Be concise. Include examples if helpful. Plain text only, no markdown, no code blocks, no backticks, no asterisks for bold/italic.`, inputContext, question)
	} else {
		prompt = fmt.Sprintf(`Answer briefly: %s

Be concise. Include examples if helpful. Plain text only, no markdown, no code blocks, no backticks, no asterisks for bold/italic.`, question)
	}

	response := callAPI(config, prompt)
	fmt.Println(response)
}

func callAPI(config *Config, prompt string) string {
	reqBody := APIRequest{
		Model: model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling request: %v\n", err)
		os.Exit(1)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making API request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "API error (status %d): %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	if len(apiResp.Choices) == 0 {
		fmt.Fprintln(os.Stderr, "No response from API")
		os.Exit(1)
	}

	return apiResp.Choices[0].Message.Content
}

func selectAndCopyCommand(response string) {
	fmt.Print("\nSelect a command (1-3): ")

	var reader *bufio.Reader
	tty, err := os.Open("/dev/tty")
	if err != nil {
		reader = bufio.NewReader(os.Stdin)
	} else {
		defer tty.Close()
		reader = bufio.NewReader(tty)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input != "1" && input != "2" && input != "3" {
		fmt.Fprintln(os.Stderr, "Invalid selection. Please enter 1, 2, or 3.")
		os.Exit(1)
	}

	lines := strings.Split(response, "\n")
	var selectedCommand string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, input+".") {
			selectedCommand = strings.TrimSpace(strings.TrimPrefix(line, input+"."))
			break
		}
	}

	if selectedCommand == "" {
		fmt.Fprintln(os.Stderr, "Could not find the selected command")
		os.Exit(1)
	}

	selectedCommand = strings.Trim(selectedCommand, "`")

	if err := copyToClipboard(selectedCommand); err != nil {
		fmt.Fprintf(os.Stderr, "Error copying to clipboard: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Command copied to clipboard: %s\n", selectedCommand)
}

func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	if _, err := exec.LookPath("xclip"); err == nil {
		cmd = exec.Command("xclip", "-selection", "clipboard")
	} else if _, err := exec.LookPath("xsel"); err == nil {
		cmd = exec.Command("xsel", "--clipboard", "--input")
	} else if _, err := exec.LookPath("wl-copy"); err == nil {
		cmd = exec.Command("wl-copy")
	} else {
		return fmt.Errorf("no clipboard utility found (install xclip, xsel, or wl-clipboard)")
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
