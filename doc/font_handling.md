# Jeppesen Font Handling

## Overview

Jeppesen chart fonts must be loaded into the Windows GDI font table before rendering charts. Without this, text will render using fallback fonts like Times New Roman.

## Font Files Location

```
C:\ProgramData\Jeppesen\Common\Fonts\
```

### File Types

1. **JTF Files** (Jeppesen TrueType Fonts)
   - Extension: `.jtf`
   - Format: Standard TrueType Font (magic: `00 01 00 00`)
   - Examples: `FONT11.JTF`, `FONT12.JTF`, `FONT13.JTF`, etc.
   - These are the main chart fonts

2. **TTF Files** (TrueType Fonts)
   - `JEPPESEN.TTF` - Main Jeppesen font
   - `JeppHeaderFont.ttf` - Header font
   - `wXstation.ttf` - Weather station font

3. **Configuration Files**
   - `jeppesen.tfl` - Font definition list (maps font IDs to names)
     - Magic: "YO" at offset 0
     - Contains: Font11, Font12, Font13, etc.
   - `jeppesen.tls` - Line style definitions
     - Magic: "ZO" at offset 0

## Font Loading Process

### Analysis from Terminal.dll

The `TclPostInit` function (at `0x1001D7C0`) loads fonts:

```c
// Pseudocode from Terminal.dll:TclPostInit
void TclPostInit(const char* fontDir) {
    // 1. Find all .jtf files in font directory
    WIN32_FIND_DATAA findData;
    HANDLE hFind = FindFirstFileA("*.jtf", &findData);
    
    // 2. Load each JTF file
    do {
        char fontPath[MAX_PATH];
        sprintf(fontPath, "%s\\%s", fontDir, findData.cFileName);
        if (AddFontResourceA(fontPath) > 0) {
            // Store path for cleanup later
            loadedFonts.Add(fontPath);
        }
    } while (FindNextFileA(hFind, &findData));
    
    // 3. Load Jeppesen.ttf explicitly
    char jeppesenPath[MAX_PATH];
    sprintf(jeppesenPath, "%s\\Jeppesen.ttf", fontDir);
    AddFontResourceA(jeppesenPath);
    
    FindClose(hFind);
}
```

### Font Creation in mrvdrv.dll

The `MF_CreateFont` function (at `0x10001540`) creates font handles:

```c
int MF_CreateFont(float height, float width, int rotation, 
                  int bold, int italic, int underline,
                  LPCSTR faceName, HFONT* outFont, int* outValid) {
    LOGFONTA lf = {0};
    lf.lfHeight = (int)height;
    lf.lfWidth = (int)width;
    lf.lfEscapement = rotation;
    lf.lfWeight = bold ? 700 : 500;
    lf.lfItalic = italic;
    lf.lfUnderline = underline;
    lf.lfCharSet = 0;  // DEFAULT_CHARSET
    lf.lfOutPrecision = 7;  // OUT_DEFAULT_PRECIS
    lf.lfQuality = 2;  // PROOF_QUALITY
    lstrcpynA(lf.lfFaceName, faceName, 31);
    
    HFONT hFont = CreateFontIndirectA(&lf);
    if (hFont) {
        *outFont = hFont;
        *outValid = 1;
        return 1;
    }
    return -1805;  // Error code
}
```

## Implementation

### Loading Fonts

```c
int LoadJeppesenFonts(const char* fontDir) {
    char searchPath[MAX_PATH];
    WIN32_FIND_DATAA findData;
    HANDLE hFind;
    int loaded = 0;
    
    // Load .jtf files
    snprintf(searchPath, MAX_PATH, "%s\\*.jtf", fontDir);
    hFind = FindFirstFileA(searchPath, &findData);
    if (hFind != INVALID_HANDLE_VALUE) {
        do {
            char fontPath[MAX_PATH];
            snprintf(fontPath, MAX_PATH, "%s\\%s", fontDir, findData.cFileName);
            if (AddFontResourceA(fontPath) > 0) {
                loaded++;
            }
        } while (FindNextFileA(hFind, &findData));
        FindClose(hFind);
    }
    
    // Load .ttf files
    snprintf(searchPath, MAX_PATH, "%s\\*.ttf", fontDir);
    hFind = FindFirstFileA(searchPath, &findData);
    if (hFind != INVALID_HANDLE_VALUE) {
        do {
            char fontPath[MAX_PATH];
            snprintf(fontPath, MAX_PATH, "%s\\%s", fontDir, findData.cFileName);
            if (AddFontResourceA(fontPath) > 0) {
                loaded++;
            }
        } while (FindNextFileA(hFind, &findData));
        FindClose(hFind);
    }
    
    return loaded;
}
```

### Unloading Fonts

```c
void UnloadJeppesenFonts(void) {
    // Store loaded paths and call RemoveFontResourceA for each
    for (int i = 0; i < gNumLoadedFonts; i++) {
        RemoveFontResourceA(gLoadedFonts[i]);
    }
}
```

## Font Face Names

The TFL file maps font IDs to face names:
- Font1, Font2, ... Font8
- Font11, Font12, Font13, ... Font84
- Font21, Font23, Font24, ... Font27
- Font32, Font34, Font35, Font43, Font52, Font65, Font79-84
- Font71, Font73-77

When `MF_CreateFont` is called with a face name like "Font11", it must match a font that was loaded via `AddFontResourceA`.

## Notes

1. Fonts must be loaded BEFORE calling `TCL_LibInit`
2. The font face name in `MF_CreateFont` must match the internal name of the loaded font
3. JTF files are just TTF files with a different extension
4. After loading, call `SendMessage(HWND_BROADCAST, WM_FONTCHANGE, 0, 0)` to notify other applications (optional)
