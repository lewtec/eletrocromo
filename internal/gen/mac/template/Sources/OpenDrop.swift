import Foundation

enum OpenDrop {
    static func cacheDir() -> URL {
        let base = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first
            ?? FileManager.default.temporaryDirectory
        let id = Bundle.main.bundleIdentifier ?? "eletrocromo"
        let dir = base.appendingPathComponent(id, isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir
    }

    static func applyEnv(_ env: inout [String: String]) {
        let support = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
            ?? FileManager.default.temporaryDirectory
        let id = Bundle.main.bundleIdentifier ?? "eletrocromo"
        let root = support.appendingPathComponent(id, isDirectory: true)
        let data = root.appendingPathComponent("data", isDirectory: true)
        let config = root.appendingPathComponent("config", isDirectory: true)
        let cache = cacheDir()
        try? FileManager.default.createDirectory(at: data, withIntermediateDirectories: true)
        try? FileManager.default.createDirectory(at: config, withIntermediateDirectories: true)
        env["ELETROCROMO_DATA_DIR"] = data.path
        env["ELETROCROMO_CACHE_DIR"] = cache.path
        env["ELETROCROMO_CONFIG_DIR"] = config.path
    }

    static func deliver(_ urls: [URL]) {
        guard !urls.isEmpty else { return }
        var urlLines: [String] = []
        var files: [String] = []
        for url in urls {
            let scheme = url.scheme?.lowercased() ?? ""
            if scheme != "http", scheme != "https", scheme != "file" {
                urlLines.append(jsonLine(kind: "url", url: url.absoluteString, paths: nil))
                continue
            }
            if let path = materialize(url) {
                files.append(path)
            }
        }
        for line in urlLines {
            append(line)
        }
        if !files.isEmpty {
            append(jsonLine(kind: "files", url: nil, paths: files))
        }
    }
}
