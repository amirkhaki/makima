import QtQuick
import Quickshell.Io
import qs.Common
import qs.Widgets
import qs.Modules.Plugins

PluginComponent {
    id: root

    property bool isConnected: false
    property var pendingPopup: null

    readonly property string statusIcon: {
        if (!isConnected) return "cloud_off"
        if (pendingPopup) return "warning"
        return "check_circle"
    }

    DankSocket {
        id: socket
        path: "/tmp/makima.sock"

        onConnectionStateChanged: {
            root.isConnected = connected
        }

        parser: SplitParser {
            onRead: line => {
                if (!line || line.length === 0) return
                try {
                    const msg = JSON.parse(line)
                    if (msg.method === "popup") {
                        root.pendingPopup = msg.params
                        popupTimer.restart()
                    }
                } catch (e) {}
            }
        }
    }

    Timer {
        id: popupTimer
        interval: 30000
        repeat: false
        onTriggered: {
            root.pendingPopup = null
        }
    }

    horizontalBarPill: Component {
        DankIcon {
            name: root.statusIcon
            color: root.isConnected ? Theme.primary : Theme.surfaceTextDim
            size: Theme.iconSizeSmall
            anchors.verticalCenter: parent.verticalCenter
        }
    }

    verticalBarPill: Component {
        DankIcon {
            name: root.statusIcon
            color: root.isConnected ? Theme.primary : Theme.surfaceTextDim
            size: Theme.iconSizeSmall
            anchors.verticalCenter: parent ? parent.verticalCenter : undefined
        }
    }

    popoutWidth: 350
    popoutHeight: 200

    popoutContent: Component {
        PopoutComponent {
            id: popout
            headerText: "Makima"
            detailsText: root.isConnected ? "Connected" : "Disconnected"
            showCloseButton: true

            Column {
                width: parent.width
                visible: root.pendingPopup !== null

                StyledText {
                    text: root.pendingPopup ? root.pendingPopup.message || "" : ""
                    color: Theme.surfaceText
                    font.pixelSize: Theme.fontSizeMedium
                    wrapMode: Text.WordWrap
                }
            }

            Column {
                width: parent.width
                visible: root.pendingPopup === null

                StyledText {
                    text: "No active rules"
                    color: Theme.surfaceTextDim
                    font.pixelSize: Theme.fontSizeMedium
                }
            }
        }
    }

    Component.onCompleted: {
        socket.connected = true
    }
}
