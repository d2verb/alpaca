package main

import (
	"fmt"
	"os"
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
	if len(presets) == 0 {
		fmt.Println("No presets available.")
		paths, err := getPaths()
		if err != nil {
			return err
		}
		fmt.Printf("Add presets to: %s\n", paths.Presets)
		return nil
	}

	fmt.Println("Available presets:")
	for _, p := range presets {
		fmt.Printf("  - %s\n", p.(string))
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
