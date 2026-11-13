import AppKit
import UniformTypeIdentifiers

struct Settings: Codable {
    var folder: String
    var autostart: Bool
}

func takeString(_ pointer: UnsafeMutablePointer<CChar>?) -> String? {
    guard let pointer else { return nil }
    defer { free(pointer) }
    return String(cString: pointer)
}

final class AppDelegate: NSObject, NSApplicationDelegate {
    private let statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
    private let menu = NSMenu()
    private var settings = Settings(folder: "", autostart: false)
    private var watching = false
    private var recentLogs: [String] = []

    func applicationDidFinishLaunching(_ notification: Notification) {
        if let json = takeString(LoadSettingsJSON()), let data = json.data(using: .utf8),
           let loaded = try? JSONDecoder().decode(Settings.self, from: data) {
            settings = loaded
        }
        statusItem.button?.image = NSImage(systemSymbolName: "eraser", accessibilityDescription: "Officeプロパティ消去")
        statusItem.menu = menu
        let launchedForWatching = CommandLine.arguments.dropFirst().contains { arg in
            arg.withCString { IsWatchOnLaunchArg(UnsafeMutablePointer(mutating: $0)) }
        }
        launchedForWatching ? startWatching() : rebuildMenu()
        Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in self?.drainLogs() }
    }

    private func startWatching() {
        settings.folder.withCString { StartWatch(UnsafeMutablePointer(mutating: $0)) }
        watching = true
        rebuildMenu()
    }

    private func stopWatching() {
        StopWatch()
        watching = false
        rebuildMenu()
    }

    private func drainLogs() {
        var changed = false
        while let message = takeString(NextLog()) {
            recentLogs.insert(message, at: 0)
            changed = true
        }
        if changed {
            recentLogs = Array(recentLogs.prefix(10))
            rebuildMenu()
        }
    }

    private func saveSettings() {
        let error = settings.folder.withCString { SaveSettings(UnsafeMutablePointer(mutating: $0), settings.autostart) }
        if let message = takeString(error) {
            recentLogs.insert("設定を保存できません: \(message)", at: 0)
        }
    }

    private func item(_ title: String, _ action: Selector?, enabled: Bool = true) -> NSMenuItem {
        let item = NSMenuItem(title: title, action: action, keyEquivalent: "")
        item.target = self
        item.isEnabled = enabled
        return item
    }

    private func rebuildMenu() {
        menu.removeAllItems()
        menu.autoenablesItems = false
        menu.addItem(item(watching ? "● 見張り中" : "○ 停止中", nil, enabled: false))
        menu.addItem(item(watching ? "見張りを止める" : "見張りを始める", #selector(toggleWatching)))
        menu.addItem(.separator())
        menu.addItem(item("見張るフォルダ: \(settings.folder)", nil, enabled: false))
        menu.addItem(item("フォルダを変更…", #selector(chooseFolder)))
        menu.addItem(item("ファイルを選んで今すぐ消す…", #selector(scrubChosen)))
        let autostart = item("ログイン時に自動で始める", #selector(toggleAutostart))
        autostart.state = settings.autostart ? .on : .off
        menu.addItem(autostart)
        menu.addItem(.separator())
        menu.addItem(item("最近の処理", nil, enabled: false))
        for log in recentLogs.isEmpty ? ["（まだありません）"] : recentLogs {
            menu.addItem(item("  " + log, nil, enabled: false))
        }
        menu.addItem(.separator())
        menu.addItem(item("終了", #selector(quit)))
    }

    @objc private func toggleWatching() {
        watching ? stopWatching() : startWatching()
    }

    @objc private func chooseFolder() {
        let panel = NSOpenPanel()
        panel.canChooseDirectories = true
        panel.canChooseFiles = false
        panel.directoryURL = URL(fileURLWithPath: settings.folder)
        NSApp.activate(ignoringOtherApps: true)
        guard panel.runModal() == .OK, let url = panel.url else { return }
        settings.folder = url.path
        saveSettings()
        if watching { startWatching() } else { rebuildMenu() }
    }

    @objc private func scrubChosen() {
        let panel = NSOpenPanel()
        panel.allowsMultipleSelection = true
        panel.allowedContentTypes = ["xlsx", "xlsm", "docx", "docm", "pptx", "pptm"].compactMap { UTType(filenameExtension: $0) }
        NSApp.activate(ignoringOtherApps: true)
        guard panel.runModal() == .OK else { return }
        for url in panel.urls {
            url.path.withCString { ScrubPath(UnsafeMutablePointer(mutating: $0)) }
        }
    }

    @objc private func toggleAutostart() {
        settings.autostart.toggle()
        saveSettings()
        rebuildMenu()
    }

    @objc private func quit() {
        NSApp.terminate(nil)
    }
}

let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.setActivationPolicy(.accessory)
app.run()
