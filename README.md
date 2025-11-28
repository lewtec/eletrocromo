# eletrocromo

A simpler approach to have desktop apps without relying on Electron or Wails.

## Architecture

This project basically deals with the web service shenanigans, so you can focus on writing the handler. On start,
the application finds an existing chromium-like installation and runs it in borderless mode to launch the app.

If no chromium-like browser is found, it uses the system way of launching URLs to open the page in the user's default browser.

