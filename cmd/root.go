/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"

	markdown "github.com/MichaelMure/go-term-markdown"
	"github.com/gdamore/tcell/v2"
	"github.com/joho/godotenv"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v3"

	link "yafai/internal/bridge/link"
	wsp "yafai/internal/bridge/wsp"

	db "yafai/internal/bridge/db"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	reflection "google.golang.org/grpc/reflection"
)

const (
	yafaiArt = `
▗▖  ▗▖▗▄▖ ▗▄▄▄▖ ▗▄▖ ▗▄▄▄▖
 ▝▚▞▘▐▌ ▐▌▐▌   ▐▌ ▐▌  █  
  ▐▌ ▐▛▀▜▌▐▛▀▀▘▐▛▀▜▌  █  
  ▐▌ ▐▌ ▐▌▐▌   ▐▌ ▐▌▗▄█▄▖
                         `
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func renderMarkdown(md string) string {
	result := markdown.Render(md, 80, 6)
	plain := ansiRegexp.ReplaceAllString(string(result), "")
	return plain
}

func StartLink(ctx context.Context) (err error) {

	lis, err := net.Listen("tcp", ":7001")
	if err != nil {
		slog.Error("failed to listen", "error", err)
		return err
	}

	l := grpc.NewServer()

	linkServer, err := link.NewLinkServer("localhost:7002")
	if err != nil {
		slog.Error("Link server creation failed.")
	}

	link.RegisterChatServiceServer(l, linkServer)
	reflection.Register(l)

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		slog.Info("Shutting down YAFAI link...")
		l.GracefulStop()
		slog.Info("Yafai Link graceful shudown complete.")
	}()

	slog.Info("YAFAI link listening on port :7001")
	if err := l.Serve(lis); err != nil {
		slog.Error("failed to start link", "error", err)
		return err
	}
	return nil
}

func StartWsp(ctx context.Context) (err error) {

	lis, err := net.Listen("tcp", ":7002")
	if err != nil {
		slog.Error("failed to listen", "error", err)
		return err
	}

	s := grpc.NewServer()
	db_path := fmt.Sprintf("%s/yafai.db", os.Getenv("YAFAI_ROOT"))
	db_client, err := db.NewDBWrapper(ctx, db_path)
	if err != nil {
		slog.Error("DB Connection failed", err.Error())
	}
	wspServer := &wsp.WorkspaceServer{
		Db: db_client,
	}
	wsp.RegisterWorkspaceServiceServer(s, wspServer)

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		slog.Info("Shutting down YAFAI Workspace...")
		s.GracefulStop()
		slog.Info("Yafai Workspace graceful shudown complete.")
	}()

	slog.Info("YAFAI link listening on port :7002")
	if err := s.Serve(lis); err != nil {
		slog.Error("failed to start link", "error", err)
		return err
	}

	return nil
}

func showWorkspaceSelection(ctx context.Context, app *tview.Application, dbw *db.DBWrapper, asciiArt string) {
	workspaces, err := dbw.ListWorkspaces(ctx)
	if err != nil {
		slog.Error("Failed to list workspaces", "error", err)
		return
	}
	if len(workspaces) == 0 {
		slog.Warn("No workspaces found")
		return
	}

	// --- Art ---
	artView := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText("[yellow]" + asciiArt + "[white]")

	// --- Welcome Message ---
	messageView := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText("[::b]Welcome to YAFAI\n\nSelect a workspace to continue.")

	// --- Workspace List ---
	list := tview.NewList().ShowSecondaryText(false)
	list.SetBorder(true).SetTitle("[::b] Workspaces [::-]").SetTitleAlign(tview.AlignCenter)
	//list.Set(40) // keep it narrow and centered
	for _, ws := range workspaces {
		wsCopy := ws // capture for closure
		list.AddItem(fmt.Sprintf("[::b]%s[::-]", wsCopy.Name), "", 0, func() {
			slog.Info("Workspace selected", "workspace", wsCopy.Name)
			layout, inputField, err := runWorkspaceUI(ctx, app, dbw, wsCopy.ID)
			if err != nil {
				slog.Error("Failed to initialize workspace UI", "error", err)
				app.Stop()
				return
			}
			app.SetRoot(layout, true).SetFocus(inputField)
			slog.Info("UI updated with workspace layout", "workspace", wsCopy.Name)
		})
	}

	// --- Footer/help bar ---
	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[::b][↑/↓] Navigate   [Enter] Select   [q] Quit[::-]").
		SetBorderPadding(0, 0, 1, 1)

		// --- Main vertical layout ---
	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(artView, 0, 10, false).
		AddItem(messageView, 3, 0, false).
		AddItem(list, 10, 0, true).
		AddItem(footer, 1, 0, false)

	// Center the whole layout vertically and horizontally
	centered := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(layout, 0, 1, true).
		AddItem(nil, 0, 1, false)

	app.SetRoot(centered, true).SetFocus(list)

	// Handle quit
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && (event.Rune() == 'q' || event.Rune() == 'Q') {
			app.Stop()
			return nil
		}
		return event
	})
}

func RunClient(ctx context.Context, dbw *db.DBWrapper) error {
	slog.Info("RunClient called")
	defer slog.Info("RunClient completed")

	app := tview.NewApplication()
	app.EnableMouse(true)
	showWorkspaceSelection(ctx, app, dbw, yafaiArt)

	// app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	// 	if event.Key() == tcell.KeyRune && (event.Rune() == 'h' || event.Rune() == 'H') {
	// 		showWorkspaceSelection(ctx, app, dbw, yafaiArt)
	// 		return nil
	// 	}
	// 	return event
	// })

	return app.Run()
}

func renderThreadMessages(chatView *tview.TextView, dbw *db.DBWrapper, ctx context.Context, threadID int64) {
	chatView.Clear()
	messages, err := dbw.ListMessagesByThread(ctx, threadID)
	if err != nil {
		chatView.Write([]byte("[red]Failed to load messages for this thread\n"))
		return
	}
	for _, msg := range messages {
		slog.Info("Rendering message-----------------------------------", "From", msg.From, "Content", msg.Content)
		slog.Info(msg.Content)
		chatView.Write([]byte(fmt.Sprintf("[yellow]%s: [white]%s\n", msg.From, msg.Content)))
		chatView.Write([]byte("[white]----------------------------------------\n"))
	}
}

func runWorkspaceUI(ctx context.Context, app *tview.Application, dbw *db.DBWrapper, workspaceID int64) (*tview.Flex, *tview.InputField, error) {
	slog.Info("runWorkspaceUI called", "workspaceID", workspaceID)
	defer func() {
		slog.Info("runWorkspaceUI completed", "workspaceID", workspaceID)
	}()
	wsp, err := dbw.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		slog.Error("Failed to load workspace", "workspaceID", workspaceID, "error", err)
		return nil, nil, fmt.Errorf("failed to load workspace: %w", err)
	}

	title := fmt.Sprintf("[yellow::b] YAFAI - %s workspace", wsp.Name)

	banner := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetRegions(true).
		SetBorder(true).
		SetTitle(fmt.Sprintf("%s   [white][\"home\"][::bu] Workspaces[::-]", title))

	threadList := tview.NewList().ShowSecondaryText(false)
	threadList.SetBorder(true).SetTitle(" Threads ")

	var layout *tview.Flex
	var statusView *tview.TextView
	var currentThread *db.Thread
	var showNewThreadDialog func()
	var chatView *tview.TextView

	statusView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	statusView.SetBorder(true).SetTitle(" Status ")

	leftColumnContainer := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(threadList, 0, 7, false).
		AddItem(statusView, 0, 3, false)

	chatView = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	chatView.SetBorder(true).SetTitle(" Workspace Chat ")
	chatView.SetDynamicColors(true)
	chatView.SetChangedFunc(func() {
		chatView.ScrollToEnd()
		app.Draw()
	})

	inputField := tview.NewInputField().
		SetLabel("You: ").
		SetFieldWidth(0)
	inputField.SetBorder(true)
	inputField.SetFieldBackgroundColor(tcell.ColorDefault)
	inputField.SetFieldTextColor(tcell.ColorWhite)

	chatContainer := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(chatView, 0, 1, false).
		AddItem(inputField, 3, 0, true)

	splitContainer := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(leftColumnContainer, 0, 3, false).
		AddItem(chatContainer, 0, 7, true)

	mainFrame := tview.NewFrame(splitContainer).SetBorders(1, 1, 2, 2, 1, 1)

	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[blue][Ctrl+C or Cmd+C] Quit[::-]").
		SetBorderPadding(0, 0, 1, 1)

	layout = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(banner, 3, 0, false).
		AddItem(mainFrame, 0, 1, true).
		AddItem(footer, 1, 0, false)

	threads, err := dbw.ListThreadsForWorkspace(ctx, workspaceID)
	if err != nil {
		slog.Error("Failed to load threads", "workspaceID", workspaceID, "error", err)
		threads = []db.Thread{}
	}
	if len(threads) > 0 && currentThread == nil {
		currentThread = &threads[0]
	}

	refreshThreads := func() {
		threadList.Clear()
		if len(threads) == 0 {
			currentThread = nil
			chatView.Clear()
			chatView.Write([]byte("[yellow]No threads found. Press 'n' or select [+] New Thread to create one.[white]\n"))
			statusView.SetText("No threads available.")
		} else {
			for i, th := range threads {
				thCopy := th
				threadList.AddItem(th.Name, "", 0, func() {
					currentThread = &thCopy
					statusView.SetText(fmt.Sprintf("Selected thread: %s", thCopy.Name))
					renderThreadMessages(chatView, dbw, ctx, thCopy.ID)
				})
				// Always set currentThread to the first thread if not set
				if i == 0 && currentThread == nil {
					currentThread = &thCopy
				}
			}
		}
		threadList.AddItem("[::b][+] New Thread", "", 0, func() {
			showNewThreadDialog()
		})
		// Always render messages for the current thread after refreshing
		if currentThread != nil {
			renderThreadMessages(chatView, dbw, ctx, currentThread.ID)
		}
	}
	refreshThreads()
	if currentThread != nil {
		renderThreadMessages(chatView, dbw, ctx, currentThread.ID)
	}

	// After refreshThreads() and before returning layout
	threadList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == 'n' {
			// Focus the last item, which is always "New Thread"
			idx := threadList.GetItemCount() - 1
			if idx >= 0 {
				threadList.SetCurrentItem(idx)
				// Optionally, trigger the action immediately:
				threadList.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), nil)
			}
			return nil // prevent further handling
		}
		return event
	})

	showNewThreadDialog = func() {
		input := tview.NewInputField().SetLabel("Title: ")
		form := tview.NewForm().
			AddFormItem(input).
			AddButton("Create", func() {
				title := input.GetText()
				if title != "" {
					thread := &db.Thread{
						Name:        title,
						WorkspaceID: workspaceID,
						Status:      "active",
					}
					err := dbw.CreateThread(ctx, thread)
					if err == nil {
						threads, _ = dbw.ListThreadsForWorkspace(ctx, workspaceID)
						if len(threads) > 0 {
							currentThread = &threads[len(threads)-1]
						}
						refreshThreads()
						if currentThread != nil {
							renderThreadMessages(chatView, dbw, ctx, currentThread.ID)
						}
					} else {
						statusView.SetText(fmt.Sprintf("Failed to create thread: %v", err))
					}
				}
				app.SetRoot(layout, true).SetFocus(threadList)
			}).
			AddButton("Cancel", func() {
				app.SetRoot(layout, true).SetFocus(threadList)
			})
		form.SetBorder(true).SetTitle("New Thread")

		// Wrap the form in a modal-like Flex for floating effect
		modal := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false). // top spacer
			AddItem(
				tview.NewFlex().
					SetDirection(tview.FlexColumn).
					AddItem(nil, 0, 2, false). // left spacer
					AddItem(form, 60, 0, true).
					AddItem(nil, 0, 2, false), // right spacer
							10, 0, true).
			AddItem(nil, 0, 1, false) // bottom spacer

		app.SetRoot(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(layout, 0, 1, false).
			AddItem(modal, 0, 1, true),
			true).SetFocus(input)
	}

	// If no threads, prompt to create one
	if len(threads) == 0 {
		// Auto-create a thread for demo
		_ = dbw.CreateThread(ctx, &db.Thread{Name: "First Thread", WorkspaceID: workspaceID, Status: "active"})
		threads, _ = dbw.ListThreadsForWorkspace(ctx, workspaceID)
		refreshThreads()
	} else {
		refreshThreads()
	}
	conn, err := grpc.NewClient("localhost:7001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Failed to connect to gRPC server", "error", err)
		return nil, nil, err
	}

	client := link.NewChatServiceClient(conn)
	stream, err := client.ChatStream(context.Background())
	if err != nil {
		slog.Error("Failed to open chat stream", "error", err)
		return nil, nil, err
	}

	inputField.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			text := inputField.GetText()
			if text != "" && currentThread != nil {
				slog.Info("Sending message", "WspId", fmt.Sprintf("%d", workspaceID), "ThreadId", fmt.Sprintf("%d", currentThread.ID))
				if err := stream.Send(&link.ChatRequest{
					WspId:    fmt.Sprintf("%d", workspaceID),
					ThreadId: fmt.Sprintf("%d", currentThread.ID),
					Request:  text,
				}); err != nil {
					slog.Error("Failed to send message", "error", err)
				}
				// --- FIX: Store your own message in the DB ---
				_ = dbw.AddMessageToThread(ctx, &db.Message{
					ThreadID: currentThread.ID,
					From:     "You",
					To:       "YAFAI",
					Content:  text,
				})
				inputField.SetText("")
				renderThreadMessages(chatView, dbw, ctx, currentThread.ID)
			}
		}
	})

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				slog.Error("Stream closed by server", err)
				break
			}
			if err != nil {
				slog.Error("Failed to receive message", "error", err)
				break
			}
			if !strings.Contains(resp.Response, "STATUS:") {
				// --- FIX: Store server message in the DB ---
				if currentThread != nil {
					_ = dbw.AddMessageToThread(ctx, &db.Message{
						ThreadID: currentThread.ID,
						From:     "YAFAI",
						To:       "You",
						Content:  resp.Response,
					})
					renderThreadMessages(chatView, dbw, ctx, currentThread.ID)
				}
			} else {
				serverMsg := resp.Response
				statusView.Write([]byte("[blue]" + renderMarkdown(serverMsg) + "\n\n"))
				statusView.Write([]byte("[white]----------------------------------------\n"))
				refreshThreads()
				if currentThread != nil {
					renderThreadMessages(chatView, dbw, ctx, currentThread.ID)
				}
			}
		}
	}()

	go func() {
		<-ctx.Done()
		slog.Info("Shutting down TUI...")
		conn.Close()
		app.Stop()

		slog.Info("Yafai TUI graceful shudown complete.")
	}()
	// Set the root of the application to the layout
	app.SetRoot(layout, true).SetFocus(inputField)
	// Force a redraw to ensure everything is displayed correctly
	return layout, inputField, nil
}

func StartYafai(env string, mode string, configsPath string) error {
	slog.Info("StartYafai called", "env", env, "mode", mode, "configsPath", configsPath)
	defer slog.Info("StartYafai completed")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Error("Failed to get user home directory", "error", err)
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	yafaiRoot := fmt.Sprintf("%s/.yafai", homeDir)
	configsDir := fmt.Sprintf("%s/configs", yafaiRoot)
	envPath := fmt.Sprintf("%s/.env", yafaiRoot)

	// 2. Ensure .yafai directory exists
	if _, err := os.Stat(yafaiRoot); os.IsNotExist(err) {
		if err := os.Mkdir(yafaiRoot, 0755); err != nil {
			slog.Error("Failed to create .yafai directory", "path", yafaiRoot, "error", err)
			return fmt.Errorf("failed to create .yafai directory: %w", err)
		}
		slog.Info("Created .yafai directory", "path", yafaiRoot)
	}

	// 3. Ensure .yafai/configs directory exists
	if _, err := os.Stat(configsDir); os.IsNotExist(err) {
		if err := os.Mkdir(configsDir, 0755); err != nil {
			slog.Error("Failed to create .yafai/configs directory", "path", configsDir, "error", err)
			return fmt.Errorf("failed to create .yafai/configs directory: %w", err)
		}
		slog.Info("Created .yafai/configs directory", "path", configsDir)
	}

	// 4. Prepare env map and read existing .env if present
	envVars := map[string]string{
		"YAFAI_ROOT": yafaiRoot,
		"GROQ_HOST":  "https://api.groq.com/openai",
	}
	if _, err := os.Stat(envPath); err == nil {
		// Load existing env file
		existing, err := godotenv.Read(envPath)
		if err == nil {
			for k, v := range existing {
				envVars[k] = v
			}
		}
	}

	// 5. Ask for GROQ_TOKEN if not present
	if envVars["GROQ_TOKEN"] == "" {
		fmt.Println(".env file created or updated. Please enter your GROQ_TOKEN:")
		fmt.Print("Enter GROQ_TOKEN: ")
		groqTokenBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			slog.Error("Error reading GROQ_TOKEN", "error", err)
			return fmt.Errorf("error reading GROQ_TOKEN: %w", err)
		}
		fmt.Println()
		envVars["GROQ_TOKEN"] = string(groqTokenBytes)
		fmt.Println("GROQ_TOKEN saved to .env file.")
	}

	// 6. Write all envVars back to .env file
	var envLines []string
	for k, v := range envVars {
		envLines = append(envLines, fmt.Sprintf("%s=%s", k, v))
	}
	if err := os.WriteFile(envPath, []byte(strings.Join(envLines, "\n")+"\n"), 0644); err != nil {
		slog.Error("Error writing .env file", "path", envPath, "error", err)
		return fmt.Errorf("error writing .env file: %w", err)
	}

	_ = godotenv.Load(envPath)

	var logPath string
	if env == "prod" {
		logPath = fmt.Sprintf("%s/yafai.log", yafaiRoot)
	} else {
		logPath = "tmp/debug.log"
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		slog.Error("Failed to open log file", "path", logPath, "error", err)
		return fmt.Errorf("failed to open log file: %v", err)
	}
	//defer logFile.Close()

	// Configure the logger to write to the log file
	logFileHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(logFileHandler)
	slog.SetDefault(logger)

	if err != nil {
		slog.Error("Error setting up YAFAI: %v", err.Error(), nil)
		os.Exit(1)
	}

	//Set root path to env
	rootPath := os.Getenv("YAFAI_ROOT")
	slog.Info("Test of slog logging")
	slog.Info("Root set to : %s", rootPath)

	if configsPath != "default" {
		slog.Info("Configs Path set to :%s", configsPath)
	} else {
		configsPath = fmt.Sprintf("%s/configs", rootPath)
		slog.Info("Configs Path set to :%s", configsPath)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Function to start a service and handle errors
	startService := func(name string, startFunc func(context.Context) error) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Panic in "+name, "error", r)
					cancel() // Cancel context on panic
				}
			}()
			err := startFunc(ctx)
			if err != nil {
				slog.Error("Error starting "+name, "error", err)
				cancel() // Cancel context on error
			} else {
				slog.Info(name + " started successfully")
			}
		}()
	}

	// Always start the link and workspace servers
	startService("YAFAI link", StartLink)
	startService("YAFAI workspace server", StartWsp)

	// If TUI mode, run the TUI client
	if mode == "tui" {
		startService("YAFAI TUI client", func(ctx context.Context) error {
			dbPath := fmt.Sprintf("%s/yafai.db", rootPath)
			dbw, err := db.NewDBWrapper(ctx, dbPath)
			if err != nil {
				slog.Error("Error opening DB for TUI", "error", err)
				return fmt.Errorf("error opening DB for TUI: %w", err)
			}
			return RunClient(ctx, dbw)
		})
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("Received signal, shutting down...", "signal", sig)
		cancel() // Context Cancel
	case <-ctx.Done():
		slog.Info("Context canceled, shutting down...")
	}

	wg.Wait()
	slog.Info("Shutdown complete.")

	return nil
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal()
	}
}

var loadCmd = &cobra.Command{
	Use:   "load [yaml-file]",
	Short: "Load a workspace from a YAML file",
	Long:  `Creates a new workspace in the database from a YAML configuration file.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		yamlFile := args[0]

		// Read YAML file
		data, err := ioutil.ReadFile(yamlFile)
		if err != nil {
			fmt.Printf("Failed to read YAML file: %v\n", err)
			os.Exit(1)
		}

		// Parse YAML into db.Workspace struct
		var ws db.Workspace
		if err := yaml.Unmarshal(data, &ws); err != nil {
			fmt.Printf("Failed to parse YAML: %v\n", err)
			os.Exit(1)
		}

		// Open DB
		rootPath := os.Getenv("YAFAI_ROOT")
		if rootPath == "" {
			homeDir, _ := os.UserHomeDir()
			rootPath = fmt.Sprintf("%s/.yafai", homeDir)
		}
		dbPath := fmt.Sprintf("%s/yafai.db", rootPath)
		ctx := context.Background()
		dbw, err := db.NewDBWrapper(ctx, dbPath)
		if err != nil {
			fmt.Printf("Failed to open DB: %v\n", err)
			os.Exit(1)
		}
		defer dbw.Close()

		// Insert agents in orchestrator.team and collect their IDs
		var agentIDs []int64
		for i := range ws.Orchestrator.Team {
			agent := &ws.Orchestrator.Team[i]
			if err := dbw.CreateAgent(ctx, agent); err != nil {
				fmt.Printf("Failed to create agent '%s': %v\n", agent.Name, err)
				os.Exit(1)
			}
			agentIDs = append(agentIDs, agent.ID)
		}

		// Create workspace
		if err := dbw.CreateWorkspace(ctx, &ws); err != nil {
			fmt.Printf("Failed to create workspace: %v\n", err)
			os.Exit(1)
		}

		// Reference agents in workspace_agents table
		for _, agentID := range agentIDs {
			if err := dbw.AddAgentToWorkspace(ctx, ws.ID, agentID); err != nil {
				fmt.Printf("Failed to reference agent %d in workspace %d: %v\n", agentID, ws.ID, err)
				os.Exit(1)
			}
		}

		fmt.Printf("Workspace '%s' created with ID %d and %d agents referenced\n", ws.Name, ws.ID, len(agentIDs))
	},
}

var rootCmd = &cobra.Command{
	Use:   "yafai",
	Short: "YAFAI-Yet Another Framwework for Agentic Interfaces",
	Long:  `Root command for YAFAI application.`,
	Run: func(cmd *cobra.Command, args []string) {
		env, _ := cmd.Flags().GetString("env")
		mode, _ := cmd.Flags().GetString("mode")
		configsPath, _ := cmd.Flags().GetString("configsPath")
		StartYafai(env, mode, configsPath)
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

func init() {
	var env string
	var mode string
	var configsPath string

	rootCmd.PersistentFlags().StringVarP(&env, "env", "e", "prod", "YAFAI env mode")
	rootCmd.Flags().StringVarP(&mode, "mode", "m", "tui", "YAFAI run mode")
	rootCmd.PersistentFlags().StringVarP(&configsPath, "configsPath", "c", "default", "Config files Path for running YAFAI workspace")

	rootCmd.AddCommand(loadCmd)

}
