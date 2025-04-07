/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"yafai/internal/nexus/workspace"

	"github.com/gdamore/tcell/v2"
	"github.com/joho/godotenv"
	"github.com/rivo/tview"

	bridge "yafai/internal/bridge"
	link "yafai/internal/bridge/proto"
	config "yafai/internal/nexus/configs"

	"github.com/spf13/cobra"
	grpc "google.golang.org/grpc"
	reflection "google.golang.org/grpc/reflection"
)

func setupYafai() (err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	yafaiRoot := fmt.Sprintf("%s/.yafai", homeDir)
	configsDir := fmt.Sprintf("%s/configs", yafaiRoot)

	// Check if .yafai directory exists, create if not
	if _, err := os.Stat(yafaiRoot); os.IsNotExist(err) {
		if err := os.Mkdir(yafaiRoot, 0755); err != nil {
			return fmt.Errorf("failed to create .yafai directory: %w", err)
		}
		slog.Info("Created .yafai directory", "path", yafaiRoot)
	}

	// Check if .yafai/configs directory exists, create if not
	if _, err := os.Stat(configsDir); os.IsNotExist(err) {
		if err := os.Mkdir(configsDir, 0755); err != nil {
			return fmt.Errorf("failed to create .yafai/configs directory: %w", err)
		}
		slog.Info("Created .yafai/configs directory", "path", configsDir)
	}

	envPath := fmt.Sprintf("%s/.env", yafaiRoot)

	// Check if the .env file exists
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// Create the .env file
		file, err := os.Create(envPath)
		if err != nil {
			return fmt.Errorf("error creating .env file: %w", err)
		}
		defer file.Close()

		fmt.Println(".env file created. Please enter your GROQ_TOKEN:")

		// Read GROQ_TOKEN from user input
		var groqToken string
		for {
			fmt.Print("Enter GROQ_TOKEN: ")
			_, err := fmt.Scanln(&groqToken)
			if err != nil || groqToken == "" {
				fmt.Println("Invalid input. Please enter a valid GROQ_TOKEN.")
				continue
			}
			break
		}

		// Write GROQ_TOKEN and GROQ_HOST to the .env file
		_, err = file.WriteString(fmt.Sprintf("GROQ_TOKEN=%s\n", groqToken))
		if err != nil {
			return fmt.Errorf("error writing to .env file: %w", err)
		}
		_, err = file.WriteString(fmt.Sprintf("GROQ_HOST=%s\n", "https://api.groq.com/openai"))
		if err != nil {
			return fmt.Errorf("error writing to .env file: %w", err)
		}
		fmt.Println("GROQ_TOKEN saved to .env file.")
	}

	err = godotenv.Load(fmt.Sprintf("%s/.env", yafaiRoot))
	if err != nil {
		slog.Error("Error loading .env file")
	}

	os.Setenv("YAFAI_ROOT", yafaiRoot)

	return err
}

func StartLink(ctx context.Context, wsp *workspace.Workspace) (err error) {

	lis, err := net.Listen("tcp", ":7001")
	if err != nil {
		slog.Error("failed to listen", "error", err)
		return err
	}

	s := grpc.NewServer()

	linkServer := &bridge.LinkServer{Wsp: wsp}

	link.RegisterOrchestratorServer(s, linkServer)
	link.RegisterPlannerServer(s, linkServer)
	link.RegisterAgentServer(s, linkServer)
	link.RegisterChatServiceServer(s, linkServer)

	reflection.Register(s)

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		slog.Info("Shutting down YAFAI link...")
		s.GracefulStop()
	}()

	slog.Info("YAFAI link listening on port :7001")
	if err := s.Serve(lis); err != nil {
		slog.Error("failed to start link", "error", err)
		return err
	}

	return nil
}

func RunClient(wsp *workspace.Workspace, logFilePath string) error {

	// Set up logging to file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Configure the logger to write to the log file
	logFileHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(logFileHandler)
	slog.SetDefault(logger)

	app := tview.NewApplication()
	title := fmt.Sprintf("[yellow::b] YAFAI - %s workspace", wsp.Name)
	// Top banner with YAFAI heading
	banner := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	banner.SetText(title).SetBorder(true)

	// Side view for traces
	sideView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	sideView.SetBorder(true).SetTitle(" System Trace ")

	// Chat view for messages
	chatView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	chatView.SetBorder(true).SetTitle(" Workspace Chat ")
	chatView.SetChangedFunc(func() {
		chatView.ScrollToEnd()
		app.Draw()
	})

	// Track width of chatView
	// var chatWidth int
	// chatView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	// 	_, _, w, _ := chatView.GetInnerRect()
	// 	chatWidth = w
	// 	return event
	// })

	// Input field for user
	inputField := tview.NewInputField().
		SetLabel("You: ").
		SetFieldWidth(0)
	inputField.SetBorder(true)
	inputField.SetFieldBackgroundColor(tcell.ColorDefault)
	inputField.SetFieldTextColor(tcell.ColorWhite)

	// Container for chat messages & input
	chatContainer := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(chatView, 0, 1, false).
		AddItem(inputField, 3, 0, true)

	// Split view (left: system trace, right: chat)
	splitContainer := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(sideView, 0, 3, false).
		AddItem(chatContainer, 0, 10, true)

	// Frame with padding and borders
	mainFrame := tview.NewFrame(splitContainer).SetBorders(1, 1, 2, 2, 1, 1)

	// Full layout (banner + main)
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(banner, 3, 0, false).
		AddItem(mainFrame, 0, 1, true)

	// gRPC connection
	conn, err := grpc.NewClient("localhost:7001", grpc.WithInsecure())
	if err != nil {
		slog.Error("Failed to connect to gRPC server", "error", err)
		return err
	}

	// Increase font size in chat view

	defer conn.Close()

	client := link.NewChatServiceClient(conn)
	stream, err := client.ChatStream(context.Background())
	if err != nil {
		slog.Error("Failed to open chat stream", "error", err)
		return err
	}

	// Handle user input
	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := inputField.GetText()
			if text != "" {
				userMsg := "You: \n" + text
				//alignedMsg := alignRight(userMsg, chatWidth)
				chatView.Write([]byte("\n[blue]" + userMsg + "\n"))
				chatView.Write([]byte("[white]----------------------------------------\n"))
				if err := stream.Send(&link.ChatRequest{Request: text}); err != nil {
					slog.Error("Failed to send message", "error", err)
				}
				inputField.SetText("")
			}
		}
	})

	// Handle server responses
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				slog.Error("Failed to receive message", "error", err)
				break
			}
			// if resp.Trace != "" {

			// 	continue
			// }
			serverMsg := "YAFAI: \n" + resp.Response
			chatView.Write([]byte("\n[green]" + serverMsg + "\n\n"))
			sideView.Write([]byte("[orange]" + resp.Trace + "\n"))
			chatView.Write([]byte("[white]----------------------------------------\n"))
		}
	}()

	// Run app
	if err := app.SetRoot(layout, true).SetFocus(inputField).EnableMouse(true).Run(); err != nil {
		slog.Error("application finished with error", "error", err)
		return err
	}

	return err
}

func StartYafai() {

	err := setupYafai()

	if err != nil {
		slog.Error("Error setting up YAFAI: %v", err.Error(), nil)
		os.Exit(1)
	}

	//Set root path to env
	rootPath := os.Getenv("YAFAI_ROOT")
	configsPath := fmt.Sprintf("%s/configs", rootPath)

	ctx, cancel := context.WithCancel(context.Background())

	var configPath string
	// Read the config files

	configs, err := config.GetAvailableConfigs(configsPath)
	if err != nil {
		slog.Error("No configs found at ~/.yafai/configs. Either create a config file at ~/.yafai/configs, or pass a specific config using --config flag ", "error", err)
		os.Exit(1)
	}

	if len(configs) == 0 {
		slog.Error("No configs found at ~/.yafai/configs. Create a config file at ~/.yafai/configs, refer to sample configs at https://github.com/YAFAI-Hub/core/samples/recipes ", "error", err)
		os.Exit(1)
	}

	var selectedConfig string
	if len(configs) == 1 {
		selectedConfig = configs[0]
		slog.Info("Using default config", "config", selectedConfig)
	} else {
		fmt.Println("Available configs:")
		for i, configName := range configs {
			fmt.Printf("[%d] %s\n", i+1, configName)
		}

		var choice int
		fmt.Println("Enter the number of the config you want to use: ")
		_, err = fmt.Scan(&choice)
		if err != nil {
			slog.Error("Failed to read input", "error", err)
			os.Exit(1)
		}

		if choice < 1 || choice > len(configs) {
			slog.Error("Invalid choice")
			os.Exit(1)
		}

		selectedConfig = configs[choice-1]
	}

	configPath = fmt.Sprintf("%s/%s", configsPath, selectedConfig)

	wsp := config.ParseConfig(configPath)
	slog.Info("Welcome to %s workspace", wsp.Name, nil)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := StartLink(ctx, wsp)
		if err != nil {
			slog.Error("Error starting YAFAI link: %v", err.Error(), nil)
			cancel()
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		logFilePath := fmt.Sprintf("%s/yafai.log", rootPath)
		err := RunClient(wsp, logFilePath)
		if err != nil {
			slog.Error("Error starting YAFAI client: %v", err.Error(), nil)
			cancel()
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		slog.Info("Received signal, shutting down...")
		cancel() // Context Cancel
	case <-ctx.Done():
		slog.Info("Context canceled, shutting down...")
	}

	wg.Wait()
	slog.Info("Shutdown complete.")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal()
	}
}

func init() {

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.yafai.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	var configPath string
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "~/.yafai/configs", "Config Path for running YAFAI workspace")
	//rootCmd.Flags().StringP("config", "c", "~/.yafai/configs", "Config Path for running YAFAI workspace")
}

var rootCmd = &cobra.Command{
	Use:   "yafai",
	Short: "YAFAI-Yet Another Framwework for Agentic Interfaces",
	Long:  `Root command for YAFAI application.`,
	Run: func(cmd *cobra.Command, args []string) {

		StartYafai()
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}
