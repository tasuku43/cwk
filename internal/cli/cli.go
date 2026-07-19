// Package cli owns command routing and presentation.
package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/app/configcmd"
	"github.com/tasuku43/cwk/internal/app/doctorcmd"
	"github.com/tasuku43/cwk/internal/app/samplecmd"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
	"github.com/tasuku43/cwk/internal/infra/commandconfig"
	"github.com/tasuku43/cwk/internal/infra/sampledata"
	"github.com/tasuku43/cwk/internal/infra/systemdoctor"
	"github.com/tasuku43/cwk/internal/infra/terminalui"
)

// CLI contains injected streams and application services.
type CLI struct {
	In      io.Reader
	Out     io.Writer
	Err     io.Writer
	Version string
	Commit  string

	baseCatalog      Catalog
	catalog          Catalog
	commandSelection *configcmd.Service
	terminal         terminalui.Opener
	doctor           *doctorcmd.Service
	samples          *samplecmd.Service
	chatwork         *chatworkcmd.Service
	chatworkAuth     *appauthn.Gate
	chatworkInitErr  error
	chatworkFactory  func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error)
}

// New builds the production CLI with a lazy PAT adapter. Help and local
// commands remain available without reading CWK_API_TOKEN; the token is
// resolved only when a Chatwork API task actually executes.
func New(in io.Reader, out, errOut io.Writer) *CLI {
	cli := newCLI(in, out, errOut, DefaultCatalog(), systemdoctor.New())
	cli.commandSelection = configcmd.New(commandconfig.NewFileStore())
	cli.chatworkFactory = func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error) {
		client, clientErr := chatworkapi.NewFromEnvironment()
		if clientErr != nil {
			return nil, nil, clientErr
		}
		return chatworkcmd.New(client), appauthn.New(client), nil
	}
	return cli
}

func (c *CLI) ensureChatwork(ctx context.Context) error {
	if c == nil {
		return fault.New(fault.KindContract, "missing_chatwork_port", "Chatwork タスクアダプターが設定されていません", false)
	}
	if c.chatwork != nil && c.chatworkAuth != nil {
		return nil
	}
	if c.chatworkInitErr != nil {
		return c.chatworkInitErr
	}
	if c.chatworkFactory == nil {
		return fault.New(fault.KindContract, "missing_chatwork_port", "Chatwork タスクアダプターが設定されていません", false)
	}
	service, gate, err := c.chatworkFactory(ctx)
	if err != nil {
		c.chatworkInitErr = err
		return err
	}
	c.chatwork = service
	c.chatworkAuth = gate
	return nil
}

func newCLI(in io.Reader, out, errOut io.Writer, catalog Catalog, inspector doctorcmd.InspectorPort) *CLI {
	return newCLIWithSamples(in, out, errOut, catalog, inspector, sampledata.New())
}

func newCLIWithSamples(
	in io.Reader,
	out, errOut io.Writer,
	catalog Catalog,
	inspector doctorcmd.InspectorPort,
	repository samplecmd.RepositoryPort,
) *CLI {
	if in == nil {
		in = strings.NewReader("")
	}
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}
	return &CLI{
		In: in, Out: out, Err: errOut,
		Version:     "dev",
		baseCatalog: catalog,
		catalog:     catalog,
		terminal:    terminalui.New(),
		doctor:      doctorcmd.New(inspector),
		samples:     samplecmd.New(repository),
	}
}

// RunContext validates global options and the catalog, resolves one command,
// and propagates the same context to the selected application boundary.
func (c *CLI) RunContext(ctx context.Context, args []string) int {
	if c == nil {
		return ExitInternal
	}
	if ctx == nil {
		return c.fail(nil, fault.New(
			fault.KindContract,
			"missing_context",
			"コマンドコンテキストが設定されていません。",
			false,
			fault.NextAction{Command: "help", Reason: "コンテキスト対応の CLI エントリーポイントから再試行してください。"},
		))
	}
	options, commandArgs, err := parseRootOptions(args)
	ctx = withErrorFormat(ctx, options.ErrorFormat)
	if err != nil {
		return c.failUsage(ctx, "invalid_root_options", err.Error(), "help", "グローバルオプションを修正してください。")
	}
	if err := ctx.Err(); err != nil {
		return c.fail(ctx, err)
	}
	baseCatalog := c.baseCatalog
	if len(baseCatalog.commands) == 0 {
		baseCatalog = c.catalog
	}
	if err := baseCatalog.Validate(); err != nil {
		return c.fail(ctx, fault.Wrap(
			fault.KindContract,
			"invalid_catalog",
			"コマンドカタログは無効です。",
			false,
			err,
			fault.NextAction{Command: "help", Reason: "ディスパッチ前にカタログを修復してください。"},
		))
	}
	if len(commandArgs) == 0 {
		return c.failUsage(ctx, "missing_command", "コマンドが必要です。", "help", "利用できるコマンドの結果を確認してください。")
	}

	commandArgs = normalizeRootAlias(commandArgs)
	activeCatalog, activeErr := c.resolveActiveCatalog(ctx, baseCatalog)
	if activeErr != nil {
		if !commandViewAlwaysInvocation(baseCatalog, commandArgs) {
			return c.fail(ctx, commandSelectionDispatchFault(activeErr))
		}
		activeCatalog, _, err = baseCatalog.ActiveView([]string{})
		if err != nil {
			return c.fail(ctx, fault.Wrap(
				fault.KindContract,
				"invalid_catalog",
				"コマンドカタログの制御領域は無効です。",
				false,
				err,
				fault.NextAction{Command: "help", Reason: "ディスパッチ前にカタログを修復してください。"},
			))
		}
	}
	c.catalog = activeCatalog
	commandArgs = normalizeTrailingHelpAlias(c.catalog, commandArgs)
	command, rest, found := c.catalog.Match(commandArgs)
	if !found {
		return c.failUsage(
			ctx,
			"unknown_command",
			fmt.Sprintf("コマンド %q は不明です。", strings.Join(commandArgs, " ")),
			"help",
			"正確なコマンドパスまたは名前空間を確認してください。",
		)
	}
	ctx = withCommandPath(ctx, command.Path)
	if len(rest) == 1 && isHelpFlag(rest[0]) {
		return c.emit(ctx, renderCommandHelp(command))
	}
	if err := ctx.Err(); err != nil {
		return c.fail(ctx, err)
	}

	intent := operation.Intent{Command: command.Path, Effect: command.Effect}
	if command.Effect == operation.EffectRead {
		if err := intent.Validate(); err != nil {
			return c.fail(ctx, fault.Wrap(
				fault.KindContract,
				"invalid_intent",
				"コマンドの意図は無効です。",
				false,
				err,
				fault.NextAction{Command: "help " + command.Path, Reason: "コマンド宣言を修復してください。"},
			))
		}
	}
	return command.handler(ctx, c, command, intent, rest)
}

func (c *CLI) resolveActiveCatalog(ctx context.Context, base Catalog) (Catalog, error) {
	if c == nil || c.commandSelection == nil {
		return base, nil
	}
	profile, configured, err := c.commandSelection.Load(ctx)
	if err != nil {
		return Catalog{}, err
	}
	enabled := make([]string, 0)
	if configured {
		enabled = normalizeLegacyCommandSelection(profile.EnabledCommands())
	} else {
		for _, command := range base.ConfigurableCommands() {
			enabled = append(enabled, command.Path)
		}
	}
	view, _, err := base.ActiveView(enabled)
	if err != nil {
		return Catalog{}, fault.Wrap(
			fault.KindInvalidInput,
			"command_selection_invalid",
			"保存済みのコマンド選択は無効です。",
			false,
			err,
		)
	}
	return view, nil
}

func commandViewAlwaysInvocation(base Catalog, args []string) bool {
	if len(args) == 0 {
		return false
	}
	if args[0] == "help" {
		_, selector, err := parseHelpArgs(args[1:])
		if err != nil {
			help, found := base.Lookup("help")
			return found && !help.Configurable
		}
		if selector == "" {
			return false
		}
		command, found := base.Lookup(selector)
		return found && !command.Configurable
	}
	command, _, found := base.Match(args)
	return found && !command.Configurable
}

func commandSelectionDispatchFault(err error) error {
	if public, ok := fault.PublicCopy(err); ok {
		next := fault.NextAction{Command: "help", Reason: "呼び出し元の準備ができたら元のコマンドを再試行してください。"}
		switch public.Code {
		case "command_selection_invalid":
			next = fault.NextAction{Command: "config", Reason: "無効なコマンド選択を明示的に置き換えてください。"}
		case "command_selection_unsafe", "command_selection_unavailable":
			next = fault.NextAction{Command: "doctor", Reason: "ローカル設定パスを復旧してから、コマンド選択の診断を確認してください。"}
		}
		return fault.New(
			public.Kind,
			public.Code,
			public.Message,
			public.Retryable,
			next,
		)
	}
	return err
}

func normalizeRootAlias(args []string) []string {
	switch args[0] {
	case "--help", "-h":
		return append([]string{"help"}, args[1:]...)
	case "--version", "-v":
		return append([]string{"version"}, args[1:]...)
	default:
		return args
	}
}

// normalizeTrailingHelpAlias maps the conventional `<selector> --help` form
// onto the catalog help task. Only selectors already proven by the catalog are
// rewritten, so an unknown namespace retains the normal unknown-command fault
// instead of changing error taxonomy inside the help parser.
func normalizeTrailingHelpAlias(catalog Catalog, args []string) []string {
	if len(args) < 2 || !isHelpFlag(args[len(args)-1]) {
		return args
	}
	selectorWords := args[:len(args)-1]
	selector := strings.Join(selectorWords, " ")
	commands, _ := catalog.Select(selector)
	if len(commands) == 0 {
		return args
	}
	normalized := make([]string, 1, len(args))
	normalized[0] = "help"
	return append(normalized, selectorWords...)
}

func isHelpFlag(value string) bool {
	return value == "--help" || value == "-h"
}

type rootOptions struct {
	ErrorFormat errorFormat
}

func parseRootOptions(args []string) (rootOptions, []string, error) {
	options := rootOptions{ErrorFormat: errorFormatText}
	seenErrorFormat := false
	index := 0
	for index < len(args) {
		argument := args[index]
		var value string
		switch {
		case argument == "--error-format":
			if index+1 >= len(args) {
				return options, nil, fmt.Errorf("--error-format には text または json が必要です")
			}
			index++
			value = args[index]
		case strings.HasPrefix(argument, "--error-format="):
			value = strings.TrimPrefix(argument, "--error-format=")
		default:
			return options, args[index:], nil
		}
		if seenErrorFormat {
			return options, nil, fmt.Errorf("--error-format は1回だけ指定できます")
		}
		parsed, err := parseErrorFormat(value)
		if err != nil {
			return options, nil, err
		}
		options.ErrorFormat = parsed
		seenErrorFormat = true
		index++
	}
	return options, args[index:], nil
}
