# eletrocromo

A simpler approach to have desktop apps without relying on Electron or Wails.

## Architecture

This project basically deals with the web service shenanigans, so you can focus on writing the handler. On start,
the application finds an existing Chromium-like installation (Helium preferred; see `SPEC.md`) and runs it in
borderless `--app` mode to launch the app.

If no Chromium-like browser is found, launch fails with an error. There is no system-default-browser fallback —
install [Helium](https://helium.computer/) or another Chromium-based browser that supports `--app`.

