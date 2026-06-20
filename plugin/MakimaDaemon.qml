import QtQuick
import Quickshell
import Quickshell.Wayland
import Quickshell.Io
import qs.Common
import qs.Widgets
import qs.Modules.Plugins

PluginComponent {
    id: root

    property bool isConnected: false
    property var pendingPopup: null

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
                        popupWindowLoader.active = true
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
            popupWindowLoader.active = false
        }
    }

    Loader {
        id: popupWindowLoader
        active: false
        sourceComponent: Component {
            PopupWindow {
                id: popupWindow
                visible: true

                WlrLayershell.layer: WlrLayer.Overlay
                WlrLayershell.exclusiveZone: -1
                WlrLayershell.namespace: "makima-popup"
                WlrLayershell.keyboardFocus: WlrKeyboardFocus.OnDemand

                width: 400
                height: popupCol.implicitHeight + Theme.spacingL * 4
                color: "transparent"

                anchors.top: true
                anchors.topMargin: 100

                StyledRect {
                    anchors.fill: parent
                    anchors.margins: 1
                    color: Theme.surfaceContainer
                    radius: Theme.cornerRadius
                    border.color: Theme.error
                    border.width: 2

                    Column {
                        id: popupCol
                        anchors.centerIn: parent
                        width: parent.width - Theme.spacingL * 4
                        spacing: Theme.spacingM

                        DankIcon {
                            name: "warning"
                            color: Theme.error
                            size: 48
                            anchors.horizontalCenter: parent.horizontalCenter
                        }

                        StyledText {
                            text: root.pendingPopup ? root.pendingPopup.title || "Warning" : ""
                            color: Theme.surfaceText
                            font.pixelSize: Theme.fontSizeLarge
                            font.bold: true
                            anchors.horizontalCenter: parent.horizontalCenter
                        }

                        StyledText {
                            text: root.pendingPopup ? root.pendingPopup.message || "" : ""
                            color: Theme.surfaceText
                            font.pixelSize: Theme.fontSizeMedium
                            wrapMode: Text.WordWrap
                            width: parent.width
                            horizontalAlignment: Text.AlignHCenter
                        }

                        StyledText {
                            text: {
                                if (!root.pendingPopup) return ""
                                var secs = Math.ceil(popupTimer.remainingTime / 1000)
                                return "Action in " + secs + "s..."
                            }
                            color: Theme.surfaceTextDim
                            font.pixelSize: Theme.fontSizeSmall
                            anchors.horizontalCenter: parent.horizontalCenter
                        }
                    }
                }

                Component.onCompleted: {
                    popupTimer.restart()
                }
            }
        }
    }

    Component.onCompleted: {
        socket.connected = true
    }
}
