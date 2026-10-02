import Foundation

extension OpenDrop {
    static func materialize(_ url: URL) -> String? {
        let inbox = cacheDir().appendingPathComponent("inbox", isDirectory: true)
        try? FileManager.default.createDirectory(at: inbox, withIntermediateDirectories: true)
        let dest = inbox.appendingPathComponent(UUID().uuidString + "-" + url.lastPathComponent)
        let scoped = url.startAccessingSecurityScopedResource()
        defer {
            if scoped { url.stopAccessingSecurityScopedResource() }
        }
        do {
            if FileManager.default.fileExists(atPath: dest.path) {
                try FileManager.default.removeItem(at: dest)
            }
            try FileManager.default.copyItem(at: url, to: dest)
            return dest.path
        } catch {
            return url.isFileURL ? url.path : nil
        }
    }

    static func append(_ line: String) {
        let file = cacheDir().appendingPathComponent("open.jsonl")
        let data = (line + "\n").data(using: .utf8) ?? Data()
        if FileManager.default.fileExists(atPath: file.path) {
            if let h = try? FileHandle(forWritingTo: file) {
                defer { try? h.close() }
                _ = try? h.seekToEnd()
                try? h.write(contentsOf: data)
                return
            }
        }
        try? data.write(to: file, options: .atomic)
    }

    static func jsonLine(kind: String, url: String?, paths: [String]?) -> String {
        var obj: [String: Any] = ["kind": kind]
        if let url { obj["url"] = url }
        if let paths { obj["paths"] = paths }
        let raw = try? JSONSerialization.data(withJSONObject: obj, options: [])
        return String(data: raw ?? Data(), encoding: .utf8) ?? ""
    }
}

extension ShareWatch {
    static func decode(_ line: String) -> [Any]? {
        guard let raw = line.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: raw) as? [String: Any]
        else { return nil }
        var items: [Any] = []
        if let text = obj["text"] as? String, !text.isEmpty { items.append(text) }
        if let url = obj["url"] as? String, let u = URL(string: url) { items.append(u) }
        if let paths = obj["paths"] as? [String] {
            for p in paths {
                items.append(URL(fileURLWithPath: p))
            }
        }
        return items.isEmpty ? nil : items
    }
}
