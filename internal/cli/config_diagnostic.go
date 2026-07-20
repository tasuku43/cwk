package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"

	"github.com/tasuku43/cwk/internal/domain/doctor"
	"github.com/tasuku43/cwk/internal/domain/fault"
)

const (
	commandSelectionFingerprintVersion  = "cwk-command-selection/v1"
	commandSelectionDoctorDetailGrammar = "state=<valid|unconfigured|invalid|unsafe|unavailable> source=<missing|saved|unknown> enabled=<count|unknown> disabled=<count|unknown> stale=<count|unknown> legacy=<count|unknown> fingerprint=<sha256:64-lowercase-hex|unavailable>"
)

func (c *CLI) commandSelectionDoctorCheck(ctx context.Context) (doctor.Check, bool, error) {
	if c == nil || c.commandSelection == nil {
		return doctor.Check{}, false, nil
	}
	state, err := c.loadCommandSelectionState(ctx)
	if err != nil {
		public, ok := fault.PublicCopy(err)
		if !ok || public.Code == "operation_canceled" {
			return doctor.Check{}, false, err
		}
		stateName, source := "unavailable", "unknown"
		switch public.Code {
		case "command_selection_invalid":
			stateName, source = "invalid", "saved"
		case "command_selection_unsafe":
			stateName = "unsafe"
		case "command_selection_unavailable":
			stateName = "unavailable"
		default:
			return doctor.Check{}, false, err
		}
		return doctor.Check{
			Name:   "command-selection",
			Status: doctor.CheckStatusFail,
			Detail: fmt.Sprintf("state=%s source=%s enabled=unknown disabled=unknown stale=unknown legacy=unknown fingerprint=unavailable", stateName, source),
		}, true, nil
	}

	base := c.baseCatalog
	if len(base.commands) == 0 {
		base = c.catalog
	}
	choices := base.ConfigurableCommands()
	if !state.configured {
		return doctor.Check{
			Name:   "command-selection",
			Status: doctor.CheckStatusWarn,
			Detail: fmt.Sprintf(
				"state=unconfigured source=missing enabled=0 disabled=%d stale=0 legacy=0 fingerprint=%s",
				len(choices), commandSelectionFingerprint(nil),
			),
		}, true, nil
	}
	enabled := selectedPathsInCatalogOrder(choices, state.enabled)
	status := doctor.CheckStatusPass
	stateName := "valid"
	if state.viewErr != nil {
		status = doctor.CheckStatusFail
		stateName = "invalid"
	} else if len(state.stale) != 0 || len(state.legacy) != 0 {
		status = doctor.CheckStatusWarn
	}
	return doctor.Check{
		Name:   "command-selection",
		Status: status,
		Detail: fmt.Sprintf(
			"state=%s source=%s enabled=%d disabled=%d stale=%d legacy=%d fingerprint=%s",
			stateName, state.source, len(enabled), len(choices)-len(enabled), len(state.stale), len(state.legacy), commandSelectionFingerprint(enabled),
		),
	}, true, nil
}

func commandSelectionFingerprint(paths []string) string {
	digest := sha256.New()
	writeFingerprintPart(digest, commandSelectionFingerprintVersion)
	for _, path := range paths {
		writeFingerprintPart(digest, path)
	}
	return "sha256:" + hex.EncodeToString(digest.Sum(nil))
}

func writeFingerprintPart(destination hash.Hash, value string) {
	_, _ = fmt.Fprintf(destination, "%d:", len(value))
	_, _ = destination.Write([]byte(value))
	_, _ = destination.Write([]byte{'\n'})
}
