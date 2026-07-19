package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tasuku43/cwk/internal/app/configcmd"
	"github.com/tasuku43/cwk/internal/app/execution"
	"github.com/tasuku43/cwk/internal/domain/commandselection"
	"github.com/tasuku43/cwk/internal/domain/fault"
	"github.com/tasuku43/cwk/internal/domain/operation"
	"github.com/tasuku43/cwk/internal/infra/terminalui"
)

const configEscapeTimeout = 75 * time.Millisecond

type commandSelectionState struct {
	configured bool
	source     string
	enabled    map[string]bool
	stale      []string
	legacy     []string
	viewErr    error
}

func runConfig(ctx context.Context, c *CLI, command CommandSpec, base operation.Intent, args []string) int {
	if len(args) != 0 {
		return c.failUsage(ctx, "invalid_arguments", "使い方: "+command.Usage(), "help config", "コマンド引数を付けずにセレクターを実行してください。")
	}

	state, repairWarning, err := c.loadCommandSelectionForConfig(ctx)
	if err != nil {
		return c.fail(ctx, err)
	}
	baseCatalog := c.baseCatalog
	if len(baseCatalog.commands) == 0 {
		baseCatalog = c.catalog
	}
	choices := baseCatalog.ConfigurableCommands()
	items := make([]configTUIItem, 0, len(choices))
	for _, choice := range choices {
		items = append(items, configTUIItem{
			Path: choice.Path, Summary: choice.Summary, Effect: choice.Effect, Enabled: state.enabled[choice.Path],
		})
	}
	always := baseCatalog.AlwaysCommands()
	alwaysPaths := make([]string, len(always))
	for index, command := range always {
		alwaysPaths[index] = command.Path
	}
	model := newConfigTUIModel(items, alwaysPaths)
	if repairWarning {
		model.notice = "保存済みの選択は無効です。Enter で置き換えるまで変更されません"
	} else if state.viewErr != nil {
		model.notice = "現在の選択は不完全です。必要なコマンド間の依存関係を有効にしてください"
	}
	original := cloneSelectionMap(state.enabled)

	if c == nil || c.terminal == nil {
		return c.fail(ctx, fault.New(fault.KindInternal, "terminal_setup_failed", "対話端末アダプターを利用できません。", true))
	}
	session, err := c.terminal.Open(c.In, c.Out)
	if err != nil {
		if errors.Is(err, terminalui.ErrNotTerminal) {
			return c.fail(ctx, fault.New(
				fault.KindUnavailable,
				"interactive_terminal_required",
				"コマンド選択には対話可能な stdin と stdout の端末が必要です。",
				false,
			))
		}
		if errors.Is(err, terminalui.ErrRestoreFailed) {
			return c.fail(ctx, fault.Wrap(fault.KindInternal, "terminal_restore_failed", "セットアップ失敗後に端末を完全には復元できませんでした。", false, err))
		}
		return c.fail(ctx, fault.Wrap(fault.KindInternal, "terminal_setup_failed", "対話端末を準備できませんでした。", true, err))
	}
	restored := false
	restore := func() error {
		if restored {
			return nil
		}
		restored = true
		if err := session.Close(); err != nil {
			return fault.Wrap(fault.KindInternal, "terminal_restore_failed", "端末を完全には復元できませんでした。", false, err)
		}
		return nil
	}
	defer func() { _ = restore() }()

	if err := c.renderConfigTUI(ctx, session, &model); err != nil {
		return c.finishConfigFault(ctx, restore, err)
	}

	var parser configTUIKeyParser
	buffer := make([]byte, 64)
	for {
		readContext := ctx
		cancelRead := func() {}
		pendingEscape := parser.escapeState != configTUIEscapeNone
		if pendingEscape {
			readContext, cancelRead = context.WithTimeout(ctx, configEscapeTimeout)
		}
		count, readErr := session.Read(readContext, buffer)
		cancelRead()
		if err := ctx.Err(); err != nil {
			return c.finishConfigFault(ctx, restore, err)
		}
		if count < 0 || count > len(buffer) {
			return c.finishConfigFault(ctx, restore, fault.Wrap(
				fault.KindInternal,
				"configuration_input_failed",
				"コマンド選択の入力を読み取れませんでした。",
				false,
				io.ErrNoProgress,
			))
		}
		if count > 0 {
			if code, done := c.applyConfigTUIKeys(ctx, command, base, baseCatalog, choices, state, original, session, &model, parser.feed(buffer[:count]), restore); done {
				return code
			}
		}
		if readErr != nil {
			if pendingEscape && parser.escapeState != configTUIEscapeNone && errors.Is(readErr, context.DeadlineExceeded) {
				if code, done := c.applyConfigTUIKeys(ctx, command, base, baseCatalog, choices, state, original, session, &model, parser.flushEscape(), restore); done {
					return code
				}
				continue
			}
			if errors.Is(readErr, io.EOF) {
				return c.finishConfigUnchanged(ctx, restore)
			}
			return c.finishConfigFault(ctx, restore, fault.Wrap(
				fault.KindInternal,
				"configuration_input_failed",
				"コマンド選択の入力を読み取れませんでした。",
				false,
				readErr,
			))
		}
		if count == 0 {
			return c.finishConfigFault(ctx, restore, fault.Wrap(
				fault.KindInternal,
				"configuration_input_failed",
				"コマンド選択の入力を読み取れませんでした。",
				false,
				io.ErrNoProgress,
			))
		}
	}
}

func (c *CLI) applyConfigTUIKeys(
	ctx context.Context,
	command CommandSpec,
	base operation.Intent,
	baseCatalog Catalog,
	choices []CommandSpec,
	state commandSelectionState,
	original map[string]bool,
	session terminalui.Session,
	model *configTUIModel,
	keys []configTUIKey,
	restore func() error,
) (int, bool) {
	changed := false
	for _, key := range keys {
		renderedForKey := false
		// A terminal read may contain several decoded keys. Repaint a prior
		// movement or toggle before accepting another selection action so the
		// next target and draft state have actually appeared on screen.
		if changed && configTUIKeyRequiresVisibleSelection(key) {
			model.notice = ""
			if err := c.renderConfigTUI(ctx, session, model); err != nil {
				return c.finishConfigFault(ctx, restore, err), true
			}
			changed = false
			renderedForKey = true
		}
		if configTUIKeyRequiresVisibleSelection(key) {
			// A frame that showed only resize guidance cannot authorize the key
			// that first observes a now-usable terminal. Repaint the exact current
			// identity and consume this key; a later input may act on that frame.
			if !model.selectionVisibleLastFrame {
				if !renderedForKey {
					if err := c.renderConfigTUI(ctx, session, model); err != nil {
						return c.finishConfigFault(ctx, restore, err), true
					}
				}
				continue
			}
			width, height, err := session.Size()
			if err != nil {
				return c.finishConfigFault(ctx, restore, fault.Wrap(fault.KindInternal, "terminal_setup_failed", "端末サイズを取得できませんでした。", true, err)), true
			}
			if !configTUISelectionVisible(*model, width, height) {
				if err := c.renderConfigTUI(ctx, session, model); err != nil {
					return c.finishConfigFault(ctx, restore, err), true
				}
				continue
			}
		}
		decision := model.apply(key)
		switch decision {
		case configTUICancel:
			return c.finishConfigUnchanged(ctx, restore), true
		case configTUIInterrupted:
			return c.finishConfigFault(ctx, restore, fault.New(
				fault.KindCanceled,
				"operation_canceled",
				"保存前にコマンド選択がキャンセルされました。",
				true,
			)), true
		case configTUISave:
			code, done := c.saveConfigSelection(ctx, command, base, baseCatalog, choices, state, original, model, restore)
			if done {
				return code, true
			}
			if err := c.renderConfigTUI(ctx, session, model); err != nil {
				return c.finishConfigFault(ctx, restore, err), true
			}
			changed = false
		default:
			if key == configTUIKeyUp || key == configTUIKeyDown || key == configTUIKeyToggle {
				changed = true
			}
		}
	}
	if changed {
		model.notice = ""
		if err := c.renderConfigTUI(ctx, session, model); err != nil {
			return c.finishConfigFault(ctx, restore, err), true
		}
	}
	return ExitOK, false
}

func configTUIKeyRequiresVisibleSelection(key configTUIKey) bool {
	switch key {
	case configTUIKeyUp, configTUIKeyDown, configTUIKeyToggle, configTUIKeySave:
		return true
	default:
		return false
	}
}

func (c *CLI) saveConfigSelection(
	ctx context.Context,
	command CommandSpec,
	base operation.Intent,
	baseCatalog Catalog,
	choices []CommandSpec,
	state commandSelectionState,
	original map[string]bool,
	model *configTUIModel,
	restore func() error,
) (int, bool) {
	enabled := model.enabledPaths()
	if _, _, err := baseCatalog.ActiveView(enabled); err != nil {
		model.notice = "保存できません。必要なコマンド間の依存関係を有効にしてください"
		return ExitOK, false
	}
	profile, err := commandselection.New(enabled)
	if err != nil {
		return c.finishConfigFault(ctx, restore, fault.Wrap(fault.KindInvalidInput, "command_selection_invalid", "選択したコマンドパスは無効です。", false, err)), true
	}
	request, err := buildConfigExecutionRequest(command, base)
	if err != nil {
		return c.finishConfigFault(ctx, restore, fault.Wrap(fault.KindContract, "invalid_mutation_contract", "コマンド選択の変更契約は無効です。", false, err)), true
	}
	if err := restore(); err != nil {
		return c.fail(ctx, err), true
	}
	err = execution.New(configcmd.ExplicitSavePolicy{Confirmed: true}).Invoke(
		ctx,
		request,
		func(actionContext context.Context, _ operation.Intent) error {
			if c == nil || c.commandSelection == nil {
				return fault.New(fault.KindUnavailable, "command_selection_unavailable", "コマンド選択設定を利用できません。", true)
			}
			return c.commandSelection.Save(actionContext, profile)
		},
	)
	if err != nil {
		if public, ok := fault.PublicCopy(err); ok && public.Code == "unclassified_mutation_outcome" {
			err = fault.New(
				fault.KindContract,
				"unclassified_mutation_outcome",
				commandSelectionUncertainMessage(commandSelectionFingerprint(enabled)),
				false,
			)
		}
		return c.fail(ctx, err), true
	}
	selected := make(map[string]bool, len(enabled))
	for _, path := range enabled {
		selected[path] = true
	}
	output := configSaveSuccessOutput(
		len(enabled),
		len(choices)-len(enabled),
		selectionChangeCount(choices, original, selected),
		len(state.stale)+len(state.legacy),
	)
	return c.emitMutationResult(ctx, output), true
}

func configSaveSuccessOutput(visible, hidden, changed, cleaned int) []byte {
	var output strings.Builder
	output.WriteString("コマンド表示を保存しました。\n")
	fmt.Fprintf(&output, "%d件を表示し、%d件を非表示にしました（%d件変更）。\n", visible, hidden, changed)
	if cleaned > 0 {
		fmt.Fprintf(&output, "古い設定を%d件整理しました。\n", cleaned)
	}
	return []byte(output.String())
}

func buildConfigExecutionRequest(command CommandSpec, base operation.Intent) (execution.Request, error) {
	if command.Agent.FixedTarget == nil || command.Agent.Mutation == nil {
		return execution.Request{}, fmt.Errorf("fixed command-selection target is missing")
	}
	intent := base
	intent.Target = operation.TargetRef{Kind: command.Agent.FixedTarget.Kind, ID: command.Agent.FixedTarget.StableID}
	intent.Impact = command.Agent.Mutation.Impact
	request := execution.Request{
		Intent: intent, ExpectedCommand: command.Path, ExpectedEffect: command.Effect,
		ExpectedTarget: intent.Target, ExpectedImpact: intent.Impact,
	}
	if intent != configcmd.Intent() {
		return execution.Request{}, fmt.Errorf("catalog and application command-selection targets differ")
	}
	return request, nil
}

func (c *CLI) renderConfigTUI(ctx context.Context, session terminalui.Session, model *configTUIModel) error {
	model.selectionVisibleLastFrame = false
	if err := ctx.Err(); err != nil {
		return err
	}
	width, height, err := session.Size()
	if err != nil {
		return fault.Wrap(fault.KindInternal, "terminal_setup_failed", "端末サイズを取得できませんでした。", true, err)
	}
	selectionVisible := configTUISelectionVisible(*model, width, height)
	frame := renderConfigTUIScreen(*model, width, height)
	frame = "\x1b[H\x1b[2J" + strings.ReplaceAll(frame, "\n", "\r\n")
	if _, err := writeOnce(c.Out, []byte(frame)); err != nil {
		return fault.Wrap(fault.KindInternal, "terminal_setup_failed", "コマンド選択画面を書き込めませんでした。", true, err)
	}
	model.selectionVisibleLastFrame = selectionVisible
	return nil
}

func (c *CLI) finishConfigUnchanged(ctx context.Context, restore func() error) int {
	if err := restore(); err != nil {
		return c.fail(ctx, err)
	}
	return c.emit(ctx, []byte("config は変更されませんでした\n"))
}

func (c *CLI) finishConfigFault(ctx context.Context, restore func() error, original error) int {
	if err := restore(); err != nil {
		return c.fail(ctx, err)
	}
	return c.fail(ctx, original)
}

func (c *CLI) loadCommandSelectionState(ctx context.Context) (commandSelectionState, error) {
	base := c.baseCatalog
	if len(base.commands) == 0 {
		base = c.catalog
	}
	if c == nil || c.commandSelection == nil {
		return commandSelectionState{}, fault.New(fault.KindUnavailable, "command_selection_unavailable", "コマンド選択設定を利用できません。", true)
	}
	profile, configured, err := c.commandSelection.Load(ctx)
	if err != nil {
		return commandSelectionState{}, err
	}
	return deriveCommandSelectionState(base, profile, configured), nil
}

func (c *CLI) loadCommandSelectionForConfig(ctx context.Context) (commandSelectionState, bool, error) {
	state, err := c.loadCommandSelectionState(ctx)
	if err == nil {
		return state, false, nil
	}
	public, ok := fault.PublicCopy(err)
	if !ok || public.Code != "command_selection_invalid" {
		return commandSelectionState{}, false, err
	}
	base := c.baseCatalog
	if len(base.commands) == 0 {
		base = c.catalog
	}
	return defaultCommandSelectionState(base, "repair"), true, nil
}

func deriveCommandSelectionState(base Catalog, profile commandselection.Profile, configured bool) commandSelectionState {
	if !configured {
		return defaultCommandSelectionState(base, "default")
	}
	state := commandSelectionState{configured: true, source: "saved", enabled: make(map[string]bool)}
	paths := normalizeLegacyCommandSelection(profile.EnabledCommands())
	for _, path := range profile.EnabledCommands() {
		if isLegacySelectableLocalCommand(path) {
			state.legacy = append(state.legacy, path)
		}
	}
	for _, path := range paths {
		command, known := base.Lookup(path)
		switch {
		case !known:
			state.stale = append(state.stale, path)
		case command.Configurable:
			state.enabled[path] = true
		}
	}
	_, _, state.viewErr = base.ActiveView(paths)
	return state
}

func normalizeLegacyCommandSelection(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		if !isLegacySelectableLocalCommand(path) {
			normalized = append(normalized, path)
		}
	}
	return normalized
}

func isLegacySelectableLocalCommand(path string) bool {
	return path == "doctor" || path == "version"
}

func defaultCommandSelectionState(base Catalog, source string) commandSelectionState {
	state := commandSelectionState{source: source, enabled: make(map[string]bool)}
	paths := make([]string, 0)
	for _, command := range base.ConfigurableCommands() {
		state.enabled[command.Path] = true
		paths = append(paths, command.Path)
	}
	_, _, state.viewErr = base.ActiveView(paths)
	return state
}

func cloneSelectionMap(values map[string]bool) map[string]bool {
	cloned := make(map[string]bool, len(values))
	for path, enabled := range values {
		cloned[path] = enabled
	}
	return cloned
}

func selectedPathsInCatalogOrder(choices []CommandSpec, selected map[string]bool) []string {
	paths := make([]string, 0, len(choices))
	for _, choice := range choices {
		if selected[choice.Path] {
			paths = append(paths, choice.Path)
		}
	}
	return paths
}

func selectionChangeCount(choices []CommandSpec, before, after map[string]bool) int {
	count := 0
	for _, choice := range choices {
		if before[choice.Path] != after[choice.Path] {
			count++
		}
	}
	return count
}
