package export

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Exporter struct {
	tclDir      string
	emfTool     string
	postProcess string
}

func NewExporter(tclDir string) *Exporter {
	return &Exporter{
		tclDir:      tclDir,
		emfTool:     "tcl2emf.exe",
		postProcess: "pdf_fixup_threshold.py",
	}
}

func (e *Exporter) getToolPath() string {
	exePath := e.emfTool
	if !filepath.IsAbs(exePath) {
		absPath, err := filepath.Abs(exePath)
		if err != nil {
			return exePath
		}
		return absPath
	}
	return exePath
}

func (e *Exporter) ExportToEMF(tclPath, emfPath string) error {
	if _, err := os.Stat(tclPath); err != nil {
		return fmt.Errorf("TCL file not found: %s", tclPath)
	}

	cmd := exec.Command(e.getToolPath(), tclPath, emfPath)
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("export failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (e *Exporter) ExportToPDFBytes(tclPath string, postProcess bool) ([]byte, error) {
	// 0. Concurrency Guard
	// If e.mu is added to your Exporter struct, uncomment the next two lines:
	// e.mu.Lock()
	// defer e.mu.Unlock()

	var timings []string
	tick := func(name string, t time.Time) time.Time {
		now := time.Now()
		d := now.Sub(t).Milliseconds()
		timings = append(timings, fmt.Sprintf("%s=%dms", name, d))
		return now
	}

	t := time.Now()
	tempDir, err := os.MkdirTemp("", "marinvent-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)
	t = tick("mkdir", t)

	pdfPath := filepath.Join(tempDir, "output.pdf")
	toolPath := e.getToolPath()
	workDir, _ := filepath.Abs(filepath.Dir(toolPath))

	// 1. Execute the tool
	cmd := exec.Command(toolPath, tclPath, pdfPath)
	cmd.Dir = workDir
	t = tick("setup", t)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tcl2emf failed: %w, output: %s", err, string(output))
	}
	t = tick("tcl2emf", t)

	// 2. Poll for a valid, unlocked, and FINALIZED PDF
	var pdfData []byte
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond) // Slightly slower ticks for disk I/O health
	defer ticker.Stop()

	success := false
PollLoop:
	for {
		select {
		case <-timeout:
			return nil, fmt.Errorf("timeout: PDF at %s never finalized (Spooler hang)", pdfPath)
		case <-ticker.C:
			info, err := os.Stat(pdfPath)
			// PDF must have a basic header (~100 bytes)
			if err != nil || info.Size() < 100 {
				continue
			}

			// A: Check for File Lock (Windows specific)
			// Try to open with Read/Write access to see if Spooler still has it
			f, err := os.OpenFile(pdfPath, os.O_RDWR, 0)
			if err != nil {
				// If we can't open it RDWR, the spooler still likely has a write lock
				continue
			}

			// B: Check for PDF Footer (Structural Integrity)
			// PDFs are read from the back. We need to find %%EOF.
			buf := make([]byte, 1024)
			fileSize := info.Size()
			offset := fileSize - 1024
			if offset < 0 {
				offset = 0
			}

			_, readErr := f.ReadAt(buf, offset)
			f.Close() // Always close immediately

			if readErr == nil || readErr == io.EOF {
				content := string(buf)
				if strings.Contains(content, "%%EOF") {
					// Valid PDF structure found!
					pdfData, err = os.ReadFile(pdfPath)
					if err == nil && len(pdfData) > 0 {
						success = true
						break PollLoop
					}
				}
			}
		}
	}
	t = tick("polling", t)

	// 3. Post-process
	if postProcess && success {
		if err := e.runPostProcess(pdfPath); err == nil {
			// Re-read finalized data after python fixup
			pdfData, _ = os.ReadFile(pdfPath)
		} else {
			fmt.Printf("Warning: post-processing failed: %v\n", err)
		}
	}
	t = tick("postprocess", t)

	log.Printf("[EXPORT] %s: %s", filepath.Base(tclPath), strings.Join(timings, " "))
	return pdfData, nil
}

// Helper to check for basic PDF magic number
func isValidPDF(data []byte) bool {
	return len(data) > 4 && string(data[:4]) == "%PDF"
}

func (e *Exporter) runPostProcess(pdfPath string) error {
	ppPath := e.postProcess
	if !filepath.IsAbs(ppPath) {
		absPath, err := filepath.Abs(ppPath)
		if err != nil {
			return err
		}
		ppPath = absPath
	}

	if _, err := os.Stat(ppPath); err != nil {
		return fmt.Errorf("post-process script not found: %s", ppPath)
	}

	cmd := exec.Command("python", ppPath, pdfPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (e *Exporter) ExportToPDF(tclPath, pdfPath string) error {
	if _, err := os.Stat(tclPath); err != nil {
		return fmt.Errorf("TCL file not found: %s", tclPath)
	}

	if filepath.Ext(pdfPath) != ".pdf" {
		pdfPath = pdfPath + ".pdf"
	}

	cmd := exec.Command(e.getToolPath(), tclPath, pdfPath)
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("export failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func (e *Exporter) ExportAll(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	files, err := os.ReadDir(e.tclDir)
	if err != nil {
		return fmt.Errorf("failed to read TCL directory: %w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if len(name) > 4 && name[len(name)-4:] == ".tcl" {
			tclPath := filepath.Join(e.tclDir, name)
			pdfPath := filepath.Join(outputDir, name[:len(name)-4]+".pdf")

			fmt.Printf("Exporting %s -> %s\n", name, pdfPath)
			if err := e.ExportToPDF(tclPath, pdfPath); err != nil {
				fmt.Printf("  Error: %v\n", err)
			}
		}
	}

	return nil
}
