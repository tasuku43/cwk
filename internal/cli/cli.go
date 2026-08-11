// Package cli owns command routing and presentation.
package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	appauthn "github.com/tasuku43/cwk/internal/app/authn"
	"github.com/tasuku43/cwk/internal/app/chatworkcmd"
	"github.com/tasuku43/cwk/internal/app/configcmd"
	"github.com/tasuku43/cwk/internal/app/doctorcmd"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
	"github.com/tasuku43/cwk/internal/infra/chatworkapi"
	"github.com/tasuku43/cwk/internal/infra/commandconfig"
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
	selectionLoaded  bool
	selectionSaved   bool
	terminal         terminalui.Opener
	doctor           *doctorcmd.Service
	chatwork         *chatworkcmd.Service
	chatworkAuth     *appauthn.Gate
	chatworkInitErr  error
	chatworkFactory  func(context.Context) (*chatworkcmd.Service, *appauthn.Gate, error)
	now              func() time.Time
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
		now:         time.Now,
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
	baseCanonicalArgs := normalizeCommandAlias(baseCatalog, commandArgs)
	staticHelpArgs, staticHelp := normalizeStaticHelpInvocation(baseCatalog, baseCanonicalArgs)
	helpFallback := false
	activeCatalog, activeErr := c.resolveActiveCatalog(ctx, baseCatalog)
	if activeErr != nil {
		if staticHelp && commandSelectionHelpFallbackAllowed(activeErr) {
			activeCatalog = baseCatalog
			commandArgs = staticHelpArgs
			helpFallback = true
		} else if !commandViewAlwaysInvocation(baseCatalog, commandArgs) {
			return c.fail(ctx, commandSelectionDispatchFault(activeErr))
		} else {
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
	}
	if helpFallback {
		ctx = withStaticHelpFallback(ctx)
	}
	c.catalog = activeCatalog
	selectionCheckArgs := baseCanonicalArgs
	if staticHelp {
		selectionCheckArgs = staticHelpArgs
	}
	if c.selectionLoaded && !c.selectionSaved && commandViewConfigurableInvocation(baseCatalog, selectionCheckArgs) {
		return c.fail(ctx, commandSelectionRequiredFault())
	}
	commandArgs = normalizeCommandAlias(c.catalog, commandArgs)
	commandArgs = normalizeTrailingHelpAlias(c.catalog, commandArgs)
	command, rest, found := c.catalog.Match(commandArgs)
	if !found {
		return c.failUsage(
			ctx,
			"unknown_command",
			unknownCommandMessage(commandArgs, c.catalog),
			"help",
			"正確なコマンドパスまたは名前空間を確認してください。",
		)
	}
	ctx = withCommandPath(ctx, command.Path)
	if len(rest) == 1 && isHelpFlag(rest[0]) {
		return c.emit(ctx, renderCommandHelp(command, c.catalog.Commands()))
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
	c.selectionLoaded = false
	c.selectionSaved = false
	if c == nil || c.commandSelection == nil {
		return base, nil
	}
	profile, configured, err := c.commandSelection.Load(ctx)
	if err != nil {
		return Catalog{}, err
	}
	c.selectionLoaded = true
	c.selectionSaved = configured
	enabled := make([]string, 0)
	if configured {
		enabled = normalizeLegacyCommandSelection(profile.EnabledCommands())
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

func commandViewConfigurableInvocation(base Catalog, args []string) bool {
	if len(args) == 0 {
		return false
	}
	if args[0] == "help" {
		_, selector, err := parseHelpArgs(args[1:])
		return err == nil && selectorContainsConfigurableCommand(base, selector)
	}
	if isHelpFlag(args[len(args)-1]) {
		return selectorContainsConfigurableCommand(base, strings.Join(args[:len(args)-1], " "))
	}
	if command, _, found := base.Match(args); found {
		return command.Configurable
	}
	return selectorContainsConfigurableCommand(base, strings.Join(args, " "))
}

func selectorContainsConfigurableCommand(base Catalog, selector string) bool {
	if selector == "" {
		return false
	}
	selector = canonicalHelpSelector(base, selector)
	commands, _ := base.Select(selector)
	for _, command := range commands {
		if command.Configurable {
			return true
		}
	}
	return false
}

func commandSelectionRequiredFault() error {
	return fault.New(
		fault.KindRejected,
		"command_selection_required",
		"初回設定が必要です。cwk config で使用するChatworkコマンドを選ぶと、エージェントに表示する候補が減り、選択ミスとトークン消費を抑えられます。",
		false,
		fault.NextAction{Command: "config", Reason: "使用するChatworkコマンドを選択してください。"},
	)
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
		command, found := base.Lookup(canonicalHelpSelector(base, selector))
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

func commandSelectionHelpFallbackAllowed(err error) bool {
	public, ok := fault.PublicCopy(err)
	if !ok {
		return false
	}
	switch public.Code {
	case "command_selection_invalid":
		return public.Kind == fault.KindInvalidInput
	case "command_selection_unsafe", "command_selection_unavailable":
		return public.Kind == fault.KindUnavailable
	default:
		return false
	}
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

// normalizeCommandAlias owns only finite compatibility spellings. A mapping is
// enabled only when its canonical target is an exact catalog command; it never
// guesses a path, reads configuration, or performs provider I/O.
func normalizeCommandAlias(catalog Catalog, args []string) []string {
	for _, command := range catalog.Commands() {
		for _, alias := range command.DeprecatedAliases {
			aliasWords := strings.Fields(alias)
			if !hasWordPrefix(args, aliasWords) {
				continue
			}
			canonicalWords := strings.Fields(command.Path)
			normalized := make([]string, 0, len(canonicalWords)+len(args)-len(aliasWords))
			normalized = append(normalized, canonicalWords...)
			return append(normalized, args[len(aliasWords):]...)
		}
	}
	return args
}

func hasWordPrefix(args, prefix []string) bool {
	if len(prefix) > len(args) {
		return false
	}
	for index := range prefix {
		if args[index] != prefix[index] {
			return false
		}
	}
	return true
}

// normalizeStaticHelpInvocation recognizes help without consulting command
// selection. It is used only to keep static help available when that local
// selection state cannot be loaded safely.
func normalizeStaticHelpInvocation(catalog Catalog, args []string) ([]string, bool) {
	if len(args) == 0 {
		return args, false
	}
	if args[0] == "help" {
		return normalizeTrailingHelpAlias(catalog, args), true
	}
	normalized := normalizeTrailingHelpAlias(catalog, args)
	if len(normalized) > 0 && normalized[0] == "help" {
		return normalized, true
	}
	return args, false
}

// normalizeTrailingHelpAlias maps a help flag anywhere after a known exact
// command, plus the conventional `<namespace> --help` form, onto the catalog
// help task. Unknown paths retain the normal unknown-command taxonomy.
func normalizeTrailingHelpAlias(catalog Catalog, args []string) []string {
	if len(args) < 2 {
		return args
	}
	if args[0] == "help" {
		if len(args) == 2 && isHelpFlag(args[1]) {
			return []string{"help", "help"}
		}
		normalized := make([]string, 0, len(args))
		normalized = append(normalized, "help")
		for _, argument := range args[1:] {
			if !isHelpFlag(argument) {
				normalized = append(normalized, argument)
			}
		}
		return normalized
	}
	if command, rest, found := catalog.Match(args); found {
		for _, argument := range rest {
			if !isHelpFlag(argument) {
				continue
			}
			normalized := []string{"help"}
			return append(normalized, strings.Fields(command.Path)...)
		}
	}
	if !isHelpFlag(args[len(args)-1]) {
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
			// Conventional root aliases are catalog commands after normalization,
			// not unknown global options. Leave them in argv for normalizeRootAlias.
			if argument == "--help" || argument == "-h" || argument == "--version" || argument == "-v" {
				return options, args[index:], nil
			}
			if strings.HasPrefix(argument, "-") {
				return options, nil, unknownRootOptionError(argument)
			}
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
