import UIKit

@main
final class AppDelegate: UIResponder, UIApplicationDelegate {
    var window: UIWindow?
    private var root: RootViewController?
    private let server = ServerProcess()

    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
    ) -> Bool {
        let root = RootViewController()
        self.root = root
        root.onRetry = { [weak self] in
            self?.restartServer()
        }
        let window = UIWindow(frame: UIScreen.main.bounds)
        window.rootViewController = root
        window.makeKeyAndVisible()
        self.window = window
        startServer()
        return true
    }

    func applicationWillTerminate(_ application: UIApplication) {
        server.stop()
    }

    private func startServer() {
        root?.showSplash(status: "Starting server…", detail: nil, error: false)
        server.start(
            onStatus: { [weak self] message in
                self?.root?.showSplash(status: message, detail: nil, error: false)
            },
            onReady: { [weak self] url in
                self?.root?.load(url)
            },
            onFailed: { [weak self] message in
                self?.root?.showSplash(status: "Server failed", detail: message, error: true)
            }
        )
    }

    private func restartServer() {
        server.stop()
        startServer()
    }
}
