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

    readonly property string statusText: {
        if (!isConnected) return "Offline"
        if (pendingPopup) return pendingPopup.message || "Action"
        return "Live"
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
        interval: pendingPopup ? (pendingPopup.duration || 30000) : 30000
        repeat: false
        onTriggered: {
            root.pendingPopup = null
        }
    }

    horizontalBarPill: Component {
        Row {
            id: content
            spacing: Theme.spacingS

            DankIcon {
                name: root.statusIcon
                color: root.isConnected ? Theme.primary : Theme.surfaceTextDim
                size: Theme.iconSizeSmall
                anchors.verticalCenter: parent.verticalCenter
            }

            StyledText {
                text: root.statusText
                color: Theme.surfaceText
                font.pixelSize: Theme.fontSizeSmall
                anchors.verticalCenter: parent.verticalCenter
            }
        }
    }

    verticalBarPill: Component {
        Row {
            id: content
            spacing: Theme.spacingS
            anchors.verticalCenter: parent ? parent.verticalCenter : undefined

            DankIcon {
                name: root.statusIcon
                color: root.isConnected ? Theme.primary : Theme.surfaceTextDim
                size: Theme.iconSizeSmall
            }

            StyledText {
                text: root.statusText
                color: Theme.surfaceText
                font.pixelSize: Theme.fontSizeSmall
            }
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
                spacing: Theme.spacingM

                visible: root.pendingPopup !== null

                StyledRect {
                    width: parent.width
                    height: popupCol.implicitHeight + Theme.spacingL * 2
                    radius: Theme.cornerRadius
                    color: Theme.surfaceContainerHigh

                    Column {
                        id: popupCol
                        anchors.fill: parent
                        anchors.margins: Theme.spacingL
                        spacing: Theme.spacingM

                        StyledText {
                            text: root.pendingPopup ? root.pendingPopup.title || "Warning" : ""
                            color: Theme.surfaceText
                            font.pixelSize: Theme.fontSizeLarge
                            font.bold: true
                        }

                        StyledText {
                            text: root.pendingPopup ? root.pendingPopup.message || "" : ""
                            color: Theme.surfaceText
                            font.pixelSize: Theme.fontSizeMedium
                            wrapMode: Text.WordWrap
                            width: parent.width
                        }

                        StyledText {
                            text: {
                                if (!root.pendingPopup) return ""
                                var secs = Math.ceil(popupTimer.remainingTime / 1000)
                                return "Action in " + secs + "s..."
                            }
                            color: Theme.surfaceTextDim
                            font.pixelSize: Theme.fontSizeSmall
                        }
                    }
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
