import QtQuick
import QtQuick.Shapes
import Quickshell
import Quickshell.Io
import qs.Common
import qs.Modals.Common
import qs.Widgets
import qs.Modules.Plugins

PluginComponent {
    id: root

    property string modalTitle: ""
    property string modalBody: ""
    property bool budgetMode: false
    property var budgetOptions: []
    property int countdownTotal: 0
    property real countdownStartTime: 0
    property real countdownElapsed: 0
    property int countdownRemaining: Math.max(0, Math.ceil(countdownTotal - countdownElapsed))
    property real countdownProgress: countdownTotal > 0 ? Math.max(0, Math.min(1, 1 - countdownElapsed / countdownTotal)) : 0

    DankSocket {
        id: socket
        path: "/tmp/makima.sock"

        onConnectionStateChanged: {
            console.log("Makima: socket connected:", connected)
            if (connected) {
                retryTimer.stop()
            }
        }

        parser: SplitParser {
            onRead: line => {
                if (!line || line.length === 0) return
                try {
                    var msg = JSON.parse(line)
                    console.log("Makima: received:", line)
                    if (msg.method === "popup") {
                        modalTitle = msg.params.title || "Warning"
                        modalBody = msg.params.message || ""

                        if (msg.params.budget) {
                            budgetMode = true
                            budgetOptions = msg.params.budget.options || [5, 15, 30]
                            countdownTotal = msg.params.budget.grace || 30
                        } else {
                            budgetMode = false
                            budgetOptions = []
                            countdownTotal = 30
                        }

                        countdownStartTime = Date.now()
                        countdownElapsed = 0
                        countdownTimer.restart()
                        modal.open()
                    }
                } catch (e) {
                    console.log("Makima: parse error:", e)
                }
            }
        }
    }

    function selectBudget(minutes) {
        socket.send(JSON.stringify({method: "budget.select", params: {minutes: minutes}}))
        modal.close()
        budgetMode = false
    }

    Timer {
        id: retryTimer
        interval: 2000
        repeat: true
        running: true
        onTriggered: {
            if (!socket.connected) {
                console.log("Makima: attempting connection")
                socket.connected = true
            }
        }
    }

    Timer {
        id: countdownTimer
        interval: 16
        repeat: true
        running: false
        onTriggered: {
            countdownElapsed = (Date.now() - countdownStartTime) / 1000
            if (countdownElapsed >= countdownTotal) {
                countdownElapsed = countdownTotal
                countdownTimer.stop()
                if (budgetMode) {
                    selectBudget(15)
                } else {
                    modal.close()
                }
            }
        }
    }

    DankModal {
        id: modal

        modalWidth: 500
        modalHeight: 450
        enableShadow: true
        closeOnEscapeKey: true
        closeOnBackgroundClick: false

        onOpened: Qt.callLater(() => modalFocusScope.forceActiveFocus())
        onDialogClosed: countdownTimer.stop()

        modalFocusScope.Keys.onPressed: event => {
            if (event.key === Qt.Key_Escape || event.key === Qt.Key_Return || event.key === Qt.Key_Enter) {
                if (budgetMode) {
                    selectBudget(15)
                } else {
                    modal.close()
                }
                event.accepted = true
            }
        }

        content: Component {
            Item {
                implicitHeight: contentColumn.implicitHeight

                Column {
                    id: contentColumn
                    anchors.left: parent.left
                    anchors.right: parent.right
                    anchors.top: parent.top
                    anchors.margins: Theme.spacingXL
                    spacing: Theme.spacingL

                    Item {
                        width: parent.width
                        height: 64

                        Rectangle {
                            width: 56
                            height: 56
                            radius: 28
                            anchors.centerIn: parent
                            color: Theme.primaryContainer

                            DankIcon {
                                name: "warning"
                                size: 28
                                color: Theme.primary
                                anchors.centerIn: parent
                            }
                        }
                    }

                    StyledText {
                        text: root.modalTitle
                        font.pixelSize: Theme.fontSizeXLarge
                        font.weight: Font.Bold
                        color: Theme.surfaceText
                        width: parent.width
                        horizontalAlignment: Text.AlignHCenter
                        wrapMode: Text.WordWrap
                    }

                    StyledText {
                        text: root.modalBody
                        font.pixelSize: Theme.fontSizeMedium
                        color: Theme.surfaceTextMedium
                        width: parent.width
                        horizontalAlignment: Text.AlignHCenter
                        wrapMode: Text.WordWrap
                        visible: text.length > 0
                        lineHeight: 1.5
                    }

                    // Countdown ring
                    Item {
                        width: parent.width
                        height: 200
                        clip: true

                        Rectangle {
                            anchors.centerIn: parent
                            width: 180
                            height: 180
                            radius: 90
                            color: Theme.surfaceContainerHigh

                            Shape {
                                id: countdownRing
                                anchors.centerIn: parent
                                width: 160
                                height: 160
                                layer.enabled: true
                                layer.samples: 8

                                ShapePath {
                                    strokeWidth: 10
                                    strokeColor: Theme.outlineVariant
                                    fillColor: "transparent"

                                    PathAngleArc {
                                        centerX: countdownRing.width / 2
                                        centerY: countdownRing.height / 2
                                        radiusX: countdownRing.width / 2 - 5
                                        radiusY: countdownRing.height / 2 - 5
                                        startAngle: -90
                                        sweepAngle: 360
                                    }
                                }

                                ShapePath {
                                    strokeWidth: 10
                                    strokeColor: Theme.primary
                                    fillColor: "transparent"
                                    capStyle: ShapePath.RoundCap

                                    PathAngleArc {
                                        centerX: countdownRing.width / 2
                                        centerY: countdownRing.height / 2
                                        radiusX: countdownRing.width / 2 - 5
                                        radiusY: countdownRing.height / 2 - 5
                                        startAngle: -90
                                        sweepAngle: 360 * root.countdownProgress
                                    }
                                }
                            }

                            StyledText {
                                anchors.centerIn: parent
                                text: root.countdownRemaining
                                font.pixelSize: Theme.fontSizeXLarge * 1.6
                                font.weight: Font.Bold
                                color: Theme.surfaceText
                            }
                        }
                    }

                    Item { height: Theme.spacingS; width: 1 }

                    // Budget options
                    Column {
                        visible: root.budgetMode
                        spacing: Theme.spacingM
                        anchors.horizontalCenter: parent.horizontalCenter

                        Row {
                            spacing: Theme.spacingM
                            anchors.horizontalCenter: parent.horizontalCenter

                            Repeater {
                                model: root.budgetOptions

                                Rectangle {
                                    width: 80
                                    height: 40
                                    radius: 20
                                    color: Theme.primary

                                    StyledText {
                                        text: modelData + "m"
                                        font.pixelSize: Theme.fontSizeMedium
                                        font.weight: Font.Medium
                                        color: Theme.primaryText
                                        anchors.centerIn: parent
                                    }

                                    MouseArea {
                                        anchors.fill: parent
                                        hoverEnabled: true
                                        cursorShape: Qt.PointingHandCursor
                                        onClicked: root.selectBudget(modelData)
                                    }
                                }
                            }
                        }

                        // Custom input row
                        Row {
                            spacing: Theme.spacingS
                            anchors.horizontalCenter: parent.horizontalCenter

                            DankTextField {
                                id: customMinutesField
                                width: 100
                                placeholderText: "min"
                                validator: IntValidator { bottom: 1; top: 180 }
                                onAccepted: {
                                    var val = parseInt(text)
                                    if (val > 0) root.selectBudget(val)
                                }
                            }

                            Rectangle {
                                width: 60
                                height: 40
                                radius: 20
                                color: Theme.primary

                                StyledText {
                                    text: "OK"
                                    font.pixelSize: Theme.fontSizeMedium
                                    font.weight: Font.Medium
                                    color: Theme.primaryText
                                    anchors.centerIn: parent
                                }

                                MouseArea {
                                    anchors.fill: parent
                                    hoverEnabled: true
                                    cursorShape: Qt.PointingHandCursor
                                    onClicked: {
                                        var val = parseInt(customMinutesField.text)
                                        if (val > 0) root.selectBudget(val)
                                    }
                                }
                            }
                        }
                    }

                    // Dismiss button (non-budget mode)
                    Rectangle {
                        visible: !root.budgetMode
                        width: 120
                        height: 40
                        radius: 20
                        anchors.horizontalCenter: parent.horizontalCenter
                        color: Theme.primary

                        StyledText {
                            text: "Dismiss"
                            font.pixelSize: Theme.fontSizeMedium
                            font.weight: Font.Medium
                            color: Theme.primaryText
                            anchors.centerIn: parent
                        }

                        MouseArea {
                            anchors.fill: parent
                            hoverEnabled: true
                            cursorShape: Qt.PointingHandCursor
                            onClicked: modal.close()
                        }
                    }

                    Item { width: 1; height: Theme.spacingS }
                }
            }
        }
    }

    horizontalBarPill: Component {
        DankIcon {
            name: "check_circle"
            color: Theme.primary
            size: Theme.iconSizeSmall
            anchors.verticalCenter: parent.verticalCenter
        }
    }

    verticalBarPill: Component {
        DankIcon {
            name: "check_circle"
            color: Theme.primary
            size: Theme.iconSizeSmall
            anchors.verticalCenter: parent ? parent.verticalCenter : undefined
        }
    }

    Component.onCompleted: {
        console.log("Makima: widget loaded, connecting to socket")
        socket.connected = true
    }
}
