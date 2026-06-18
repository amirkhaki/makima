import QtQuick
import qs.Common
import qs.Widgets
import qs.Modules.Plugins

PluginSettings {
    pluginId: "makima"

    StringSetting {
        settingKey: "socketPath"
        label: "Daemon Socket Path"
        description: "Path to the makima daemon Unix socket"
        placeholder: "/tmp/makima.sock"
        defaultValue: "/tmp/makima.sock"
    }

    ToggleSetting {
        settingKey: "browserTracking"
        label: "Browser Tracking"
        description: "Enable Chrome CDP browser tracking"
        defaultValue: true
    }

    ToggleSetting {
        settingKey: "hyprlandTracking"
        label: "Hyprland Tracking"
        description: "Enable Hyprland IPC tracking"
        defaultValue: true
    }
}
