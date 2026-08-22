import UIKit
import WebKit

final class RootViewController: UIViewController, WKNavigationDelegate, WKUIDelegate {
    var onRetry: (() -> Void)?

    private let webView: WKWebView
    private let refresh = UIRefreshControl()
    private let splash = UIView()
    private let statusLabel = UILabel()
    private let detailLabel = UILabel()
    private let spinner = UIActivityIndicatorView(style: .large)
    private let retryButton = UIButton(type: .system)
    private var appURL: URL?

    init() {
        let config = WKWebViewConfiguration()
        webView = WKWebView(frame: .zero, configuration: config)
        super.init(nibName: nil, bundle: nil)
        webView.navigationDelegate = self
        webView.uiDelegate = self
        if #available(iOS 16.4, *) {
            webView.isInspectable = true
        }
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemBackground

        webView.translatesAutoresizingMaskIntoConstraints = false
        webView.scrollView.alwaysBounceVertical = true
        webView.scrollView.contentInsetAdjustmentBehavior = .automatic
        refresh.addTarget(self, action: #selector(pullToRefresh), for: .valueChanged)
        refresh.accessibilityLabel = "Reload"
        webView.scrollView.refreshControl = refresh

        splash.translatesAutoresizingMaskIntoConstraints = false
        splash.backgroundColor = .systemBackground

        statusLabel.translatesAutoresizingMaskIntoConstraints = false
        statusLabel.textAlignment = .center
        statusLabel.font = .preferredFont(forTextStyle: .headline)
        statusLabel.adjustsFontForContentSizeCategory = true
        statusLabel.text = "Starting…"

        detailLabel.translatesAutoresizingMaskIntoConstraints = false
        detailLabel.textAlignment = .center
        detailLabel.font = .preferredFont(forTextStyle: .footnote)
        detailLabel.textColor = .secondaryLabel
        detailLabel.numberOfLines = 6
        detailLabel.adjustsFontForContentSizeCategory = true
        detailLabel.isHidden = true

        spinner.translatesAutoresizingMaskIntoConstraints = false

        retryButton.translatesAutoresizingMaskIntoConstraints = false
        retryButton.setTitle("Retry", for: .normal)
        retryButton.addTarget(self, action: #selector(retryTapped), for: .touchUpInside)
        retryButton.isHidden = true
        retryButton.accessibilityLabel = "Retry"

        view.addSubview(webView)
        view.addSubview(splash)
        splash.addSubview(statusLabel)
        splash.addSubview(detailLabel)
        splash.addSubview(spinner)
        splash.addSubview(retryButton)

        NSLayoutConstraint.activate([
            webView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            webView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            webView.topAnchor.constraint(equalTo: view.topAnchor),
            webView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            splash.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            splash.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            splash.topAnchor.constraint(equalTo: view.topAnchor),
            splash.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            statusLabel.centerXAnchor.constraint(equalTo: splash.centerXAnchor),
            statusLabel.centerYAnchor.constraint(equalTo: splash.centerYAnchor, constant: -12),
            statusLabel.leadingAnchor.constraint(greaterThanOrEqualTo: splash.leadingAnchor, constant: 24),
            statusLabel.trailingAnchor.constraint(lessThanOrEqualTo: splash.trailingAnchor, constant: -24),
            detailLabel.topAnchor.constraint(equalTo: statusLabel.bottomAnchor, constant: 8),
            detailLabel.leadingAnchor.constraint(equalTo: splash.leadingAnchor, constant: 32),
            detailLabel.trailingAnchor.constraint(equalTo: splash.trailingAnchor, constant: -32),
            spinner.bottomAnchor.constraint(equalTo: statusLabel.topAnchor, constant: -16),
            spinner.centerXAnchor.constraint(equalTo: splash.centerXAnchor),
            retryButton.topAnchor.constraint(equalTo: detailLabel.bottomAnchor, constant: 16),
            retryButton.centerXAnchor.constraint(equalTo: splash.centerXAnchor),
        ])
    }

    func showSplash(status: String, detail: String?, error: Bool) {
        endRefresh()
        splash.isHidden = false
        webView.isHidden = true
        statusLabel.text = status
        if let detail, !detail.isEmpty {
            detailLabel.text = detail
            detailLabel.isHidden = false
        } else {
            detailLabel.text = ""
            detailLabel.isHidden = true
        }
        if error {
            spinner.stopAnimating()
            spinner.isHidden = true
            retryButton.isHidden = false
        } else {
            spinner.isHidden = false
            spinner.startAnimating()
            retryButton.isHidden = true
        }
    }

    func load(_ url: URL) {
        appURL = url
        showSplash(status: "Loading…", detail: nil, error: false)
        webView.load(URLRequest(url: url))
    }

    func reload() {
        if let url = appURL {
            webView.load(URLRequest(url: url))
        } else {
            webView.reload()
        }
    }

    @objc private func retryTapped() {
        onRetry?()
    }

    @objc private func pullToRefresh() {
        reload()
    }

    private func endRefresh() {
        if refresh.isRefreshing {
            refresh.endRefreshing()
        }
    }

    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        splash.isHidden = true
        webView.isHidden = false
        endRefresh()
    }

    func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        endRefresh()
        showSplash(status: "Load failed", detail: error.localizedDescription, error: true)
    }

    func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
        endRefresh()
        showSplash(status: "Load failed", detail: error.localizedDescription, error: true)
    }

    func webView(
        _ webView: WKWebView,
        decidePolicyFor navigationAction: WKNavigationAction,
        decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
    ) {
        guard let url = navigationAction.request.url else {
            decisionHandler(.cancel)
            return
        }
        if Self.isLoopback(url) {
            decisionHandler(.allow)
            return
        }
        let scheme = url.scheme?.lowercased() ?? ""
        if scheme == "http" || scheme == "https" {
            UIApplication.shared.open(url)
        }
        decisionHandler(.cancel)
    }

    func webView(
        _ webView: WKWebView,
        createWebViewWith configuration: WKWebViewConfiguration,
        for navigationAction: WKNavigationAction,
        windowFeatures: WKWindowFeatures
    ) -> WKWebView? {
        if let url = navigationAction.request.url, !Self.isLoopback(url) {
            let scheme = url.scheme?.lowercased() ?? ""
            if scheme == "http" || scheme == "https" {
                UIApplication.shared.open(url)
            }
        }
        return nil
    }

    private static func isLoopback(_ url: URL) -> Bool {
        let host = url.host?.lowercased() ?? ""
        return host == "127.0.0.1" || host == "localhost" || host == "::1"
    }
}
