# Marinvent TCL DLL Reverse Engineering

## Project Goal
Reverse engineer the Marinvent tcl .dll libraries and write a program that can load a .tcl file and export it to a printer job.

## Important DLLs
- **mrvtcl.dll** (port 8193): Core TCL parsing and rendering library
- **mrvdrv.dll** (port 8194): Low-level painting driver (GDI abstraction)
- **Terminal.dll** (port 8195): Application layer using mrvtcl.dll

## Ghidra Instances
- mrvtcl.dll: `localhost:8195`
- mrvdrv.dll: `localhost:8194`
- Terminal.dll: `localhost:8193`

## Ghidra HTTP API Usage
Use `ghydra` CLI or direct HTTP requests to interact with Ghidra.

### Common Commands
```bash
# List functions
ghydra --port 8193 functions list

# Decompile a function
ghydra --port 8193 functions decompile --name <function_name>

# Get function details
ghydra --port 8193 functions get --address 0x<address>

# List exports
ghydra --port 8193 functions list --name-contains "export"
```

### Direct HTTP API
```bash
# Get program info
curl http://localhost:8193/program

# List functions
curl http://localhost:8193/functions

# Get strings
curl http://localhost:8193/strings
```

## Findings

### mrvtcl.dll (Port 8193)
- **Status**: Analyzed
- **Purpose**: Terminal Chart Library - parses and renders TCL chart files
- **Key Exports** (50 functions):
  - Initialization: `TCL_LibInit`, `TCL_LibClose`
  - File Operations: `TCL_Open`, `TCL_ClosePict`, `TCL_CloseAllPicts`
  - Picture Info: `TCL_GetNumPictsInFile`, `TCL_GetPictName`, `TCL_GetPictRect`, `TCL_GetVisibleRect`
  - Display: `TCL_Display`, `TCL_DisplayEx`
  - Groups: `TCL_GetGroupList`, `TCL_ShowGroup`, `TCL_HighlightGroup`
  - Geographic: `TCL_GeoLatLon2XY`, `TCL_GeoXY2LatLon`, `TCL_IsPictGeoRefd`
  - Palette: `TCL_SetPalette`, `TCL_GetPaletteHandle`, `TCL_GetNumColors`
  - Rotation: `TCL_Rotate`, `TCL_RotateLeftRight`
- **Calling Convention**: __cdecl (32-bit)
- **Image Base**: 0x10000000
- **Detailed API**: See `doc/mrvtcl_api.md`

### mrvdrv.dll (Port 8194)
- **Status**: Analyzed
- **Purpose**: Marinvent Framework Driver - GDI abstraction layer
- **Key Exports** (53 functions):
  - Initialization: `MF_LibOpen`, `MF_LibClose`
  - Painting Context: `MF_BeginPainting`, `MF_EndPainting`
  - Object Creation: `MF_CreatePen`, `MF_CreateBrush`, `MF_CreateFont`, `MF_CreatePalette`
  - Drawing: `MF_MoveTo`, `MF_LineTo`, `MF_DrawArc`, `MF_DrawEllipse`, `MF_DrawText`, etc.
  - Raster: `MF_LoadRaster`, `MF_PaintRaster`, `MF_CreateCompatibleRaster`
  - File I/O: `MF_OpenFile`, `MF_Read`, `MF_CloseFile`
  - Compression: `MF_DecompressData`
- **Calling Convention**: __cdecl (32-bit)
- **Image Base**: 0x10000000
- **Detailed API**: See `doc/mrvdrv_api.md`

### Terminal.dll (Port 8195)
- **Status**: Analyzed
- **Purpose**: Application framework using mrvtcl.dll
- **Key Functions**:
  - `DoTripKitPrinting`: Main printing function
  - `IPrinting`: Printing interface
  - `CPrintableView`, `RichPrintView`: MFC printing views
- **Imports**: Uses MRVTCL.dll for TCL rendering

### TCL File Format
- **Magic Bytes**: "OX" (0x4f58) at offset 0
- **Version**: Byte 2 indicates version (must be > 3)
- **Flags**: Byte 3 has flags (bit 7 = encryption/compression)
- **Checksum**: 4-byte CRC at end of data
- **Structure**:
  - Header (6 bytes): Magic + Version + Flags + Reserved
  - Picture Directory: List of picture names and offsets
  - Picture Data: Compressed chart data
- **Pictures**: A TCL file can contain multiple "pictures" (charts)

## Printing Workflow

To print a TCL file to a printer or export as EMF/PDF:

1. Initialize libraries:
   ```c
   MF_LibOpen();
   TCL_LibInit(0, 0, 0, NULL);
   ```

2. Open TCL file:
   ```c
   uint fileHandle;
   int result = TCL_Open(&fileHandle, 0, NULL, NULL);  // Get count
   uint numPicts;
   TCL_GetNumPictsInFile(fileHandle, &numPicts);
   ```

3. Select picture:
   ```c
   void* pictHandle;
   TCL_Open(&fileHandle, 1, NULL, &pictHandle);  // Open first picture
   ```

4. Create printer/metafile DC:
   ```c
   HDC hdc = CreateEnhMetaFile(...);  // Or printer DC
   ```

5. Render:
   ```c
   MF_BeginPainting(hdc);
   TCL_Display(pictHandle, hdc, 1.0f, 1.0f, NULL, NULL, 0xFFFF);
   MF_EndPainting(hdc);
   ```

6. Cleanup:
   ```c
   TCL_ClosePict(pictHandle);
   TCL_LibClose();
   MF_LibClose();
   ```

## Task Progress
- [x] Analyze mrvtcl.dll exports and key functions
- [x] Analyze mrvdrv.dll exports and key functions
- [x] Analyze Terminal.dll for reference usage
- [x] Understand TCL file format (partial)
- [x] Create 32-bit test program (tcl2emf.exe)
- [x] Fix font loading for proper chart rendering
- [x] Test with sample TCL files (LIRZ111.tcl, ELLX114.tcl)
- [x] Verify PDF/EMF output with correct fonts

## Font Loading (Critical for Proper Rendering)

### Problem
PDFs exported with Times New Roman instead of Jeppesen fonts because the fonts
weren't loaded into the Windows GDI font table before rendering.

### Solution ✓ VERIFIED
Load Jeppesen fonts using `AddFontResourceA()` before rendering:

```c
// Font location: C:\ProgramData\Jeppesen\Common\Fonts\
// Load all .jtf and .ttf files
AddFontResourceA("C:\\ProgramData\\Jeppesen\\Common\\Fonts\\JEPPESEN.TTF");
AddFontResourceA("C:\\ProgramData\\Jeppesen\\Common\\Fonts\\FONT11.JTF");
// ... etc for all font files
```

### Font Files
- **Location**: `C:\ProgramData\Jeppesen\Common\Fonts\`
- **JTF files**: Jeppesen TrueType Fonts (45 fonts total)
  - These are standard TTF files with .jtf extension
  - Magic bytes: `00 01 00 00` (TrueType signature)
- **TTF files**: JEPPESEN.TTF, JeppHeaderFont.ttf, wXstation.ttf
- **TFL file**: jeppesen.tfl - Font definition list (maps font IDs to names)
- **TLS file**: jeppesen.tls - Line style definitions

### Implementation in tcl2emf.cpp
- `LoadJeppesenFonts()` - Loads all .jtf/.ttf files from font directory
- `UnloadJeppesenFonts()` - Cleanup with RemoveFontResourceA
- Called before TCL_LibInit, unloaded at program exit
- Broadcasts WM_FONTCHANGE after loading

### Fallback Paths
The program looks for resources in this order:
1. **Local directory** (current working directory)
2. **Jeppesen installation paths**:
   - DLLs: `C:\Program Files (x86)\Jeppesen\JeppView for Windows\`
   - Fonts/config: `C:\ProgramData\Jeppesen\Common\Fonts\`

TCL chart files must be provided manually (no fallback - typically extracted from installation).

### Test Results
- LIRZ111.tcl → LIRZ111.pdf ✓
- ELLX114.tcl → ELLX114.pdf ✓
- Fonts render correctly with Jeppesen typefaces

## Current Issue
~Resolved: Font loading was the key to proper rendering.~
~Resolved: Font loading was the key to proper rendering.~

The converter now works correctly:
- Loads 45 Jeppesen fonts from local directory or `C:\ProgramData\Jeppesen\Common\Fonts\`
- Exports TCL charts to PDF and EMF formats
- Fonts render correctly with proper Jeppesen typefaces
