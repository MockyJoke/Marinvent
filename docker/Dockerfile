FROM ubuntu:24.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    g++-mingw-w64-i686 \
    golang-go \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY docs ./docs
COPY internal ./internal
COPY pdf_fixup_threshold.py ./
COPY jdmtool ./jdmtool
COPY tcl2emf.cpp ./
COPY georef_tool.cpp ./
COPY win_deps ./win_deps
COPY docker/entrypoint.sh ./docker/entrypoint.sh

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/marinvent-api ./cmd/server && \
    i686-w64-mingw32-g++ -O2 -DNDEBUG -static-libgcc -static-libstdc++ \
        tcl2emf.cpp -o /out/tcl2emf.exe -lgdi32 -luser32 -lwinspool && \
    i686-w64-mingw32-g++ -O2 -DNDEBUG -static-libgcc -static-libstdc++ \
        georef_tool.cpp -o /out/georef_tool.exe -luser32 -lgdi32 -ladvapi32 -lwinspool && \
    mkdir -p /image-root/app/win/lib /image-root/app/win/fonts /image-root/app/data/Charts /image-root/var/lib/marinvent && \
    install -m 0755 /out/marinvent-api /image-root/app/marinvent-api && \
    install -m 0755 /out/tcl2emf.exe /image-root/app/win/lib/tcl2emf.exe && \
    install -m 0755 /out/georef_tool.exe /image-root/app/win/lib/georef_tool.exe && \
    cp -r win_deps/fonts/. /image-root/app/win/fonts/ && \
    cp win_deps/lib/*.dll /image-root/app/win/lib/ && \
    install -m 0755 docker/entrypoint.sh /image-root/app/docker-entrypoint.sh && \
    cp -r jdmtool /image-root/app/jdmtool && \
    install -m 0644 pdf_fixup_threshold.py /image-root/app/pdf_fixup_threshold.py

FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

RUN dpkg --add-architecture i386 && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
        cups \
        cups-client \
        ghostscript \
        libcups2t64:i386 \
        printer-driver-cups-pdf \
        python3 \
        wine \
        wine32 \
        xvfb \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /image-root/ /

EXPOSE 8080

VOLUME ["/app/data/Charts"]

ENTRYPOINT ["/app/docker-entrypoint.sh"]
