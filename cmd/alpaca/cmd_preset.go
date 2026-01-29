package main

import (
	"fmt"
	"os"

	"github.com/d2verb/alpaca/internal/ui"
)

type PresetCmd struct {
	List   PresetListCmd `cmd:"" name:"ls" help:"List available presets"`
	Remove PresetRmCmd   `cmd:"" name:"rm" help:"Remove a preset"`
}

type PresetListCmd struct{}

func (c *PresetListCmd) Run() error {
	cl, err := newClient()
	if err != nil {
		return err
	}

	resp, err := cl.ListPresets()
	if err != nil {
		return errDaemonNotRunning()
	}

	if resp.Status == "error" {
		return fmt.Errorf("%s", resp.Error)
	}

	presets, _ := resp.Data["presets"].([]any)

	// Convert to string slice
	presetNames := make([]string, len(presets))
	for i, p := range presets {
		presetNames[i] = p.(string)
	}

	ui.PrintPresetList(presetNames)

	if len(presetNames) == 0 {
		paths, err := getPaths()
		if err != nil {
			return err
		}
		ui.PrintInfo(fmt.Sprintf("Add presets to: %s", paths.Presets))
	}

	return nil
}

type PresetRmCmd struct {
	Name string `arg:"" help:"Preset name to remove"`
}

func (c *PresetRmCmd) Run() error {
	paths, err := getPaths()
	if err != nil {
		return err
	}

	presetPath := fmt.Sprintf("%s/%s.yaml", paths.Presets, c.Name)

	// Check if preset exists
	if _, err := os.Stat(presetPath); os.IsNotExist(err) {
		return errPresetNotFound(c.Name)
	}

	// Confirmation prompt
	fmt.Printf("Delete preset '%s'? (y/N): ", c.Name)
	var response string
	fmt.Scanln(&response)
	if response != "y" && response != "Y" {
		fmt.Println("Cancelled.")
		return nil
	}

	// Delete file
	if err := os.Remove(presetPath); err != nil {
		return fmt.Errorf("remove preset: %w", err)
	}

	fmt.Printf("Preset '%s' removed.\n", c.Name)
	return nil
}
