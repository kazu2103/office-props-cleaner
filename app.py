import json
import plistlib
import queue
import subprocess
import sys
import threading
import tkinter as tk
from pathlib import Path
from tkinter import filedialog, scrolledtext

from office_props_cleaner import TARGET_SUFFIXES, try_scrub, watch

APP_NAME = "Officeプロパティ消去"
APP_ID = "jp.ascend.office-props-cleaner"
SETTINGS_PATH = Path.home() / ".office-props-cleaner.json"


def load_settings():
    try:
        return json.loads(SETTINGS_PATH.read_text(encoding="utf-8"))
    except (FileNotFoundError, ValueError):
        return {"folder": str(Path.home() / "Downloads"), "autostart": False}


def save_settings(settings):
    SETTINGS_PATH.write_text(json.dumps(settings, ensure_ascii=False), encoding="utf-8")


def launch_command():
    if getattr(sys, "frozen", False):
        return [sys.executable]
    return [sys.executable, str(Path(__file__).resolve())]


def set_autostart(enabled: bool):
    if sys.platform == "darwin":
        plist = Path.home() / "Library" / "LaunchAgents" / f"{APP_ID}.plist"
        if enabled:
            plist.parent.mkdir(parents=True, exist_ok=True)
            plist.write_bytes(plistlib.dumps({"Label": APP_ID, "ProgramArguments": launch_command(), "RunAtLoad": True}))
        else:
            plist.unlink(missing_ok=True)
    elif sys.platform == "win32":
        import winreg

        key = winreg.OpenKey(winreg.HKEY_CURRENT_USER, r"Software\Microsoft\Windows\CurrentVersion\Run", 0, winreg.KEY_SET_VALUE)
        with key:
            if enabled:
                winreg.SetValueEx(key, APP_ID, 0, winreg.REG_SZ, subprocess.list2cmdline(launch_command()))
            else:
                try:
                    winreg.DeleteValue(key, APP_ID)
                except FileNotFoundError:
                    pass


class App:
    def __init__(self, root: tk.Tk):
        self.root = root
        self.settings = load_settings()
        self.messages = queue.Queue()
        self.stop_event = threading.Event()
        self.worker = None

        root.title(APP_NAME)
        root.geometry("560x380")
        # ウィンドウを閉じても見張りを続けたいので、閉じるボタンは最小化にする
        root.protocol("WM_DELETE_WINDOW", root.iconify)

        folder_row = tk.Frame(root)
        folder_row.pack(fill="x", padx=12, pady=(12, 4))
        tk.Label(folder_row, text="見張るフォルダ:").pack(side="left")
        self.folder_label = tk.Label(folder_row, text=self.settings["folder"], anchor="w")
        self.folder_label.pack(side="left", fill="x", expand=True)
        tk.Button(folder_row, text="変更…", command=self.choose_folder).pack(side="right")

        control_row = tk.Frame(root)
        control_row.pack(fill="x", padx=12, pady=4)
        self.toggle_button = tk.Button(control_row, text="見張りを始める", width=16, command=self.toggle)
        self.toggle_button.pack(side="left")
        self.status = tk.Label(control_row, text="停止中", fg="gray")
        self.status.pack(side="left", padx=8)
        tk.Button(control_row, text="ファイルを選んで今すぐ消す…", command=self.scrub_chosen).pack(side="right")

        self.autostart = tk.BooleanVar(value=self.settings["autostart"])
        tk.Checkbutton(root, text="パソコンにログインしたら自動で見張りを始める", variable=self.autostart, command=self.toggle_autostart).pack(anchor="w", padx=12)

        self.log_area = scrolledtext.ScrolledText(root, height=12, state="disabled")
        self.log_area.pack(fill="both", expand=True, padx=12, pady=(4, 4))
        tk.Button(root, text="終了", command=self.quit).pack(anchor="e", padx=12, pady=(0, 12))

        self.start()
        self.root.after(300, self.drain_messages)

    def log(self, message):
        self.messages.put(message)

    def drain_messages(self):
        while not self.messages.empty():
            self.log_area.configure(state="normal")
            self.log_area.insert("end", self.messages.get() + "\n")
            self.log_area.see("end")
            self.log_area.configure(state="disabled")
        self.root.after(300, self.drain_messages)

    def start(self):
        # 止めた直前の見張りがまだ待機中でも動き出さないよう、毎回新しい停止フラグを使う
        self.stop_event = threading.Event()
        folder = Path(self.settings["folder"])
        self.worker = threading.Thread(target=watch, args=(folder, self.log, self.stop_event.is_set), daemon=True)
        self.worker.start()
        self.toggle_button.configure(text="見張りを止める")
        self.status.configure(text="見張り中", fg="green")

    def stop(self):
        self.stop_event.set()
        self.toggle_button.configure(text="見張りを始める")
        self.status.configure(text="停止中", fg="gray")
        self.log("見張りを止めました")

    def toggle(self):
        self.stop() if self.worker and not self.stop_event.is_set() else self.start()

    def choose_folder(self):
        chosen = filedialog.askdirectory(initialdir=self.settings["folder"])
        if not chosen:
            return
        self.settings["folder"] = chosen
        save_settings(self.settings)
        self.folder_label.configure(text=chosen)
        if not self.stop_event.is_set():
            self.stop()
            self.start()

    def scrub_chosen(self):
        patterns = " ".join(f"*{s}" for s in sorted(TARGET_SUFFIXES))
        for path in filedialog.askopenfilenames(filetypes=[("Office ファイル", patterns)]):
            try_scrub(Path(path), self.log)

    def toggle_autostart(self):
        enabled = self.autostart.get()
        try:
            set_autostart(enabled)
        except OSError as e:
            self.log(f"自動起動の設定に失敗しました: {e}")
            self.autostart.set(not enabled)
            return
        self.settings["autostart"] = enabled
        save_settings(self.settings)
        self.log("ログイン時に自動で始めます" if enabled else "自動起動をやめました")

    def quit(self):
        self.stop_event.set()
        self.root.destroy()


if __name__ == "__main__":
    root = tk.Tk()
    App(root)
    root.mainloop()
