/*
 * TCL Georeferencing Tool
 * 
 * Provides georeferencing operations for Jeppesen TCL terminal chart files.
 * 
 * Build: build-georef.bat (uses Visual Studio 32-bit compiler)
 * Usage: georef_tool <command> <tcl_file> [args...]
 * 
 * Commands:
 *   status <tcl_file>                  - Check if chart is georeferenced
 *   bounds <tcl_file>                  - Get chart pixel bounds
 *   coord2pixel <tcl_file> <lat> <lon> - Convert lat/lon to pixel coords
 *   pixel2coord <tcl_file> <x> <y>     - Convert pixel coords to lat/lon
 * 
 * Output: JSON format
 */

#include <windows.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define JEPPVIEW_PATH "C:\\Program Files (x86)\\Jeppesen\\JeppView for Windows"
#define JEPPESEN_FONTS_PATH "C:\\ProgramData\\Jeppesen\\Common\\Fonts"

typedef int (__cdecl *TCL_LibInit_t)(int, int, int, void*);
typedef int (__cdecl *TCL_LibClose_t)(void);
typedef int (__cdecl *TCL_Open_t)(const char*, unsigned int, const char*, void**);
typedef int (__cdecl *TCL_ClosePict_t)(void*);
typedef unsigned int (__cdecl *TCL_GetNumPictsInFile_t)(const char*, unsigned int*);
typedef int (__cdecl *TCL_GetPictRect_t)(void*, RECT*);
typedef int (__cdecl *TCL_IsPictGeoRefd_t)(void*);
typedef int (__cdecl *TCL_GeoLatLon2XY_t)(void*, double, double, int*, int*);
typedef int (__cdecl *TCL_GeoXY2LatLon_t)(void*, int, int, double*, double*);

typedef void (__cdecl *MF_LibOpen_t)(void);
typedef void (__cdecl *MF_LibClose_t)(void);

static HMODULE hMrvDrv = NULL;
static HMODULE hMrvTcl = NULL;

static TCL_LibInit_t TCL_LibInit = NULL;
static TCL_LibClose_t TCL_LibClose = NULL;
static TCL_Open_t TCL_Open = NULL;
static TCL_ClosePict_t TCL_ClosePict = NULL;
static TCL_GetNumPictsInFile_t TCL_GetNumPictsInFile = NULL;
static TCL_GetPictRect_t TCL_GetPictRect = NULL;
static TCL_IsPictGeoRefd_t TCL_IsPictGeoRefd = NULL;
static TCL_GeoLatLon2XY_t TCL_GeoLatLon2XY = NULL;
static TCL_GeoXY2LatLon_t TCL_GeoXY2LatLon = NULL;

static MF_LibOpen_t MF_LibOpen = NULL;
static MF_LibClose_t MF_LibClose = NULL;

static HMODULE TryLoadLibrary(const char* name) {
    HMODULE hMod = LoadLibraryA(name);
    if (hMod) return hMod;
    
    char path[MAX_PATH];
    snprintf(path, MAX_PATH, "%s\\%s", JEPPVIEW_PATH, name);
    hMod = LoadLibraryA(path);
    return hMod;
}

static int LoadDLLs(void) {
    hMrvDrv = TryLoadLibrary("mrvdrv.dll");
    if (!hMrvDrv) return 0;
    
    hMrvTcl = TryLoadLibrary("mrvtcl.dll");
    if (!hMrvTcl) {
        FreeLibrary(hMrvDrv);
        return 0;
    }
    
    MF_LibOpen = (MF_LibOpen_t)GetProcAddress(hMrvDrv, "MF_LibOpen");
    MF_LibClose = (MF_LibClose_t)GetProcAddress(hMrvDrv, "MF_LibClose");
    
    TCL_LibInit = (TCL_LibInit_t)GetProcAddress(hMrvTcl, "TCL_LibInit");
    TCL_LibClose = (TCL_LibClose_t)GetProcAddress(hMrvTcl, "TCL_LibClose");
    TCL_Open = (TCL_Open_t)GetProcAddress(hMrvTcl, "TCL_Open");
    TCL_ClosePict = (TCL_ClosePict_t)GetProcAddress(hMrvTcl, "TCL_ClosePict");
    TCL_GetNumPictsInFile = (TCL_GetNumPictsInFile_t)GetProcAddress(hMrvTcl, "TCL_GetNumPictsInFile");
    TCL_GetPictRect = (TCL_GetPictRect_t)GetProcAddress(hMrvTcl, "TCL_GetPictRect");
    TCL_IsPictGeoRefd = (TCL_IsPictGeoRefd_t)GetProcAddress(hMrvTcl, "TCL_IsPictGeoRefd");
    TCL_GeoLatLon2XY = (TCL_GeoLatLon2XY_t)GetProcAddress(hMrvTcl, "TCL_GeoLatLon2XY");
    TCL_GeoXY2LatLon = (TCL_GeoXY2LatLon_t)GetProcAddress(hMrvTcl, "TCL_GeoXY2LatLon");
    
    if (!MF_LibOpen || !TCL_LibInit || !TCL_Open || !TCL_IsPictGeoRefd) {
        return 0;
    }
    
    return 1;
}

static void UnloadDLLs(void) {
    if (hMrvTcl) FreeLibrary(hMrvTcl);
    if (hMrvDrv) FreeLibrary(hMrvDrv);
    hMrvTcl = hMrvDrv = NULL;
}

static char gLoadedFonts[100][MAX_PATH];
static int gNumLoadedFonts = 0;

static int LoadJeppesenFonts(const char* fontDir) {
    char searchPath[MAX_PATH];
    WIN32_FIND_DATAA findData;
    HANDLE hFind;
    int loaded = 0;
    
    snprintf(searchPath, MAX_PATH, "%s\\*.jtf", fontDir);
    hFind = FindFirstFileA(searchPath, &findData);
    if (hFind != INVALID_HANDLE_VALUE) {
        do {
            char fontPath[MAX_PATH];
            snprintf(fontPath, MAX_PATH, "%s\\%s", fontDir, findData.cFileName);
            if (AddFontResourceExA(fontPath, FR_PRIVATE, 0) > 0) {
                if (gNumLoadedFonts < 100) {
                    strncpy(gLoadedFonts[gNumLoadedFonts], fontPath, MAX_PATH - 1);
                    gNumLoadedFonts++;
                }
                loaded++;
            }
        } while (FindNextFileA(hFind, &findData));
        FindClose(hFind);
    }
    
    snprintf(searchPath, MAX_PATH, "%s\\*.ttf", fontDir);
    hFind = FindFirstFileA(searchPath, &findData);
    if (hFind != INVALID_HANDLE_VALUE) {
        do {
            char fontPath[MAX_PATH];
            snprintf(fontPath, MAX_PATH, "%s\\%s", fontDir, findData.cFileName);
            if (AddFontResourceExA(fontPath, FR_PRIVATE, 0) > 0) {
                if (gNumLoadedFonts < 100) {
                    strncpy(gLoadedFonts[gNumLoadedFonts], fontPath, MAX_PATH - 1);
                    gNumLoadedFonts++;
                }
                loaded++;
            }
        } while (FindNextFileA(hFind, &findData));
        FindClose(hFind);
    }
    
    return loaded;
}

static void UnloadJeppesenFonts(void) {
    for (int i = 0; i < gNumLoadedFonts; i++) {
        RemoveFontResourceExA(gLoadedFonts[i], FR_PRIVATE, 0);
    }
    gNumLoadedFonts = 0;
}

static int gInitialized = 0;

static int InitTCLLib(void) {
    if (gInitialized) return 1;
    
    char fontPath[MAX_PATH] = {0};
    char lineStylePath[MAX_PATH] = {0};
    char tclClassPath[MAX_PATH] = {0};
    char fontDir[MAX_PATH] = {0};
    
    FILE* f = fopen("jeppesen.tfl", "r");
    if (f) {
        fclose(f);
        GetFullPathNameA("jeppesen.tfl", MAX_PATH, fontPath, NULL);
        GetFullPathNameA("jeppesen.tls", MAX_PATH, lineStylePath, NULL);
        GetFullPathNameA("lssdef.tcl", MAX_PATH, tclClassPath, NULL);
        strcpy(fontDir, ".");
    } else {
        strcpy(fontDir, JEPPESEN_FONTS_PATH);
        snprintf(fontPath, MAX_PATH, "%s\\jeppesen.tfl", JEPPESEN_FONTS_PATH);
        snprintf(lineStylePath, MAX_PATH, "%s\\jeppesen.tls", JEPPESEN_FONTS_PATH);
        snprintf(tclClassPath, MAX_PATH, "%s\\lssdef.tcl", JEPPESEN_FONTS_PATH);
    }
    
    LoadJeppesenFonts(fontDir);
    MF_LibOpen();
    
    int result = TCL_LibInit((int)fontPath, (int)lineStylePath, (int)tclClassPath, NULL);
    if (result == 1) {
        gInitialized = 1;
    }
    return result == 1;
}

static void* OpenPict(const char* tclFile, int pictIndex) {
    char absPath[MAX_PATH];
    GetFullPathNameA(tclFile, MAX_PATH, absPath, NULL);
    
    void* pictHandle = NULL;
    int result = TCL_Open(absPath, pictIndex, NULL, &pictHandle);
    if (result != 1 || !pictHandle) {
        return NULL;
    }
    return pictHandle;
}

static void PrintJsonString(const char* s) {
    printf("\"");
    while (*s) {
        switch (*s) {
            case '"': printf("\\\""); break;
            case '\\': printf("\\\\"); break;
            case '\n': printf("\\n"); break;
            case '\r': printf("\\r"); break;
            case '\t': printf("\\t"); break;
            default: putchar(*s);
        }
        s++;
    }
    printf("\"");
}

static void CmdStatus(const char* tclFile) {
    void* pict = OpenPict(tclFile, 1);
    if (!pict) {
        printf("{\"error\": \"Failed to open TCL file\"}\n");
        return;
    }
    
    int geoRefd = TCL_IsPictGeoRefd(pict);
    RECT rect = {0};
    TCL_GetPictRect(pict, &rect);
    
    printf("{\"georeferenced\": %s", geoRefd == 1 ? "true" : "false");
    printf(", \"bounds\": {");
    printf("\"left\": %ld, \"top\": %ld, \"right\": %ld, \"bottom\": %ld", 
           rect.left, rect.top, rect.right, rect.bottom);
    printf(", \"width\": %ld, \"height\": %ld", 
           rect.right - rect.left, rect.bottom - rect.top);
    printf("}}\n");
    
    TCL_ClosePict(pict);
}

static void CmdCoord2Pixel(const char* tclFile, double lat, double lon) {
    void* pict = OpenPict(tclFile, 1);
    if (!pict) {
        printf("{\"error\": \"Failed to open TCL file\"}\n");
        return;
    }
    
    int geoRefd = TCL_IsPictGeoRefd(pict);
    if (geoRefd != 1) {
        printf("{\"error\": \"Chart is not georeferenced\"}\n");
        TCL_ClosePict(pict);
        return;
    }
    
    int x = 0, y = 0;
    int result = TCL_GeoLatLon2XY(pict, lat, lon, &x, &y);
    
    if (result == 1) {
        printf("{\"x\": %d, \"y\": %d}\n", x, y);
    } else if (result == -23) {
        printf("{\"error\": \"Coordinates out of chart bounds\"}\n");
    } else {
        printf("{\"error\": \"Conversion failed\", \"code\": %d}\n", result);
    }
    
    TCL_ClosePict(pict);
}

static void CmdPixel2Coord(const char* tclFile, int x, int y) {
    void* pict = OpenPict(tclFile, 1);
    if (!pict) {
        printf("{\"error\": \"Failed to open TCL file\"}\n");
        return;
    }
    
    int geoRefd = TCL_IsPictGeoRefd(pict);
    if (geoRefd != 1) {
        printf("{\"error\": \"Chart is not georeferenced\"}\n");
        TCL_ClosePict(pict);
        return;
    }
    
    double lat = 0, lon = 0;
    int result = TCL_GeoXY2LatLon(pict, x, y, &lat, &lon);
    
    if (result == 1) {
        printf("{\"latitude\": %.10f, \"longitude\": %.10f}\n", lat, lon);
    } else if (result == -23) {
        printf("{\"error\": \"Pixel coordinates out of chart bounds\"}\n");
    } else {
        printf("{\"error\": \"Conversion failed\", \"code\": %d}\n", result);
    }
    
    TCL_ClosePict(pict);
}

static void PrintUsage(const char* progName) {
    fprintf(stderr, "TCL Georeferencing Tool\n\n");
    fprintf(stderr, "Usage: %s <command> <tcl_file> [args...]\n\n", progName);
    fprintf(stderr, "Commands:\n");
    fprintf(stderr, "  status <tcl_file>                  Check if georeferenced and get bounds\n");
    fprintf(stderr, "  coord2pixel <tcl_file> <lat> <lon> Convert lat/lon to pixel coords\n");
    fprintf(stderr, "  pixel2coord <tcl_file> <x> <y>     Convert pixel coords to lat/lon\n");
}

int main(int argc, char* argv[]) {
    if (argc < 3) {
        PrintUsage(argv[0]);
        return 1;
    }
    
    const char* cmd = argv[1];
    const char* tclFile = argv[2];
    
    FILE* f = fopen(tclFile, "rb");
    if (!f) {
        printf("{\"error\": \"File not found\"}\n");
        return 1;
    }
    fclose(f);
    
    if (!LoadDLLs()) {
        printf("{\"error\": \"Failed to load DLLs\"}\n");
        return 1;
    }
    
    if (!InitTCLLib()) {
        printf("{\"error\": \"Failed to initialize TCL library\"}\n");
        UnloadDLLs();
        return 1;
    }
    
    if (strcmp(cmd, "status") == 0) {
        CmdStatus(tclFile);
    } else if (strcmp(cmd, "coord2pixel") == 0) {
        if (argc < 5) {
            printf("{\"error\": \"Missing lat/lon arguments\"}\n");
        } else {
            double lat = atof(argv[3]);
            double lon = atof(argv[4]);
            CmdCoord2Pixel(tclFile, lat, lon);
        }
    } else if (strcmp(cmd, "pixel2coord") == 0) {
        if (argc < 5) {
            printf("{\"error\": \"Missing x/y arguments\"}\n");
        } else {
            int x = atoi(argv[3]);
            int y = atoi(argv[4]);
            CmdPixel2Coord(tclFile, x, y);
        }
    } else {
        printf("{\"error\": \"Unknown command\"}\n");
    }
    
    TCL_LibClose();
    UnloadJeppesenFonts();
    UnloadDLLs();
    
    return 0;
}
