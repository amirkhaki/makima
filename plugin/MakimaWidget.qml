import QtQuick
import Quickshell.Io
import qs.Common
import qs.Widgets
import qs.Modules.Plugins

PluginComponent {
    id: root

    property var popoutService: null
    property bool isConnected: false
    property var status: ({})
    property var rules: []
    property var categories: ({})
    property var todos: []

    property var _pendingRequests: ({})
    property int _nextId: 1

    readonly property string browserCategory: {
        if (!status || !status.browser) return ""
        return status.browser.category || ""
    }

    readonly property string browserUrl: {
        if (!status || !status.browser) return ""
        return status.browser.url || ""
    }

    readonly property string workspace: {
        if (!status || !status.hyprland) return ""
        return status.hyprland.workspace || ""
    }

    readonly property string statusIcon: {
        if (!isConnected) return "cloud_off"
        if (browserCategory) return "category"
        return "check_circle"
    }

    readonly property string statusText: {
        if (!isConnected) return "Disconnected"
        if (browserCategory) return browserCategory
        return "Connected"
    }

    DankSocket {
        id: socket
        path: "/tmp/makima.sock"

        onConnectionStateChanged: {
            root.isConnected = connected
            if (connected) {
                requestStatus()
                requestRules()
                requestCategories()
                requestTodos()
            }
        }

        parser: SplitParser {
            onRead: line => {
                if (!line || line.length === 0) return
                try {
                    const response = JSON.parse(line)
                    handleResponse(response)
                } catch (e) {
                    console.error("Failed to parse response:", e)
                }
            }
        }
    }

    function _sendRequest(method, params) {
        const id = _nextId++
        const req = {method: method, id: id}
        if (params !== undefined && params !== null) {
            req.params = params
        }
        _pendingRequests[id] = method
        socket.send(req)
    }

    function requestStatus() {
        _sendRequest("status")
    }

    function requestRules() {
        _sendRequest("rule.list")
    }

    function requestCategories() {
        _sendRequest("category.list")
    }

    function requestTodos() {
        _sendRequest("todo.list")
    }

    function handleResponse(response) {
        if (response.error) {
            console.error("Daemon error:", response.error)
            return
        }

        const method = _pendingRequests[response.id]
        if (method) {
            delete _pendingRequests[response.id]

            switch (method) {
            case "status":
                root.status = response.result
                break
            case "rule.list":
                root.rules = response.result
                break
            case "category.list":
                root.categories = response.result
                break
            case "todo.list":
                root.todos = response.result
                break
            }
        }
    }

    horizontalBarPill: Component {
        Row {
            id: content
            spacing: Theme.spacingS

            DankIcon {
                name: root.statusIcon
                color: root.isConnected ? Theme.primary : Theme.surfaceTextDim
                size: 14
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
        Column {
            id: content
            spacing: Theme.spacingS

            DankIcon {
                name: root.statusIcon
                color: root.isConnected ? Theme.primary : Theme.surfaceTextDim
                size: 14
                anchors.horizontalCenter: parent.horizontalCenter
            }

            StyledText {
                text: root.statusText
                color: Theme.surfaceText
                font.pixelSize: Theme.fontSizeSmall
                rotation: 90
                anchors.horizontalCenter: parent.horizontalCenter
            }
        }
    }

    popoutWidth: 400
    popoutHeight: 500

    popoutContent: Component {
        PopoutComponent {
            id: popout

            headerText: "Makima"
            detailsText: root.isConnected ? "Connected to daemon" : "Disconnected"
            showCloseButton: true

            Column {
                width: parent.width
                spacing: Theme.spacingM

                StyledRect {
                    width: parent.width
                    height: dashboardColumn.implicitHeight + Theme.spacingL * 2
                    radius: Theme.cornerRadius
                    color: Theme.surfaceContainerHigh

                    Column {
                        id: dashboardColumn
                        anchors.fill: parent
                        anchors.margins: Theme.spacingL
                        spacing: Theme.spacingS

                        StyledText {
                            text: "Dashboard"
                            color: Theme.surfaceText
                            font.pixelSize: Theme.fontSizeLarge
                            font.bold: true
                        }

                        Row {
                            spacing: Theme.spacingS
                            StyledText {
                                text: "URL:"
                                color: Theme.surfaceText
                                font.pixelSize: Theme.fontSizeSmall
                            }
                            StyledText {
                                text: root.browserUrl || "N/A"
                                color: Theme.surfaceText
                                font.pixelSize: Theme.fontSizeSmall
                                elide: Text.ElideRight
                                width: parent.parent.width - 60
                            }
                        }

                        Row {
                            spacing: Theme.spacingS
                            StyledText {
                                text: "Category:"
                                color: Theme.surfaceText
                                font.pixelSize: Theme.fontSizeSmall
                            }
                            StyledText {
                                text: root.browserCategory || "N/A"
                                color: Theme.surfaceText
                                font.pixelSize: Theme.fontSizeSmall
                            }
                        }

                        Row {
                            spacing: Theme.spacingS
                            StyledText {
                                text: "Workspace:"
                                color: Theme.surfaceText
                                font.pixelSize: Theme.fontSizeSmall
                            }
                            StyledText {
                                text: root.workspace || "N/A"
                                color: Theme.surfaceText
                                font.pixelSize: Theme.fontSizeSmall
                            }
                        }
                    }
                }

                StyledRect {
                    width: parent.width
                    height: rulesColumn.implicitHeight + Theme.spacingL * 2
                    radius: Theme.cornerRadius
                    color: Theme.surfaceContainerHigh
                    visible: root.rules.length > 0

                    Column {
                        id: rulesColumn
                        anchors.fill: parent
                        anchors.margins: Theme.spacingL
                        spacing: Theme.spacingS

                        StyledText {
                            text: "Rules (" + root.rules.length + ")"
                            color: Theme.surfaceText
                            font.pixelSize: Theme.fontSizeLarge
                            font.bold: true
                        }

                        Repeater {
                            model: root.rules

                            Row {
                                spacing: Theme.spacingS
                                StyledText {
                                    text: modelData.name || modelData.id || "Rule"
                                    color: Theme.surfaceText
                                    font.pixelSize: Theme.fontSizeSmall
                                    elide: Text.ElideRight
                                    width: popout.width - Theme.spacingL * 2 - 40
                                }
                                StyledText {
                                    text: modelData.enabled !== false ? "ON" : "OFF"
                                    color: modelData.enabled !== false ? Theme.primary : Theme.surfaceTextDim
                                    font.pixelSize: Theme.fontSizeSmall
                                }
                            }
                        }

                        StyledText {
                            text: root.rules.length === 0 ? "No rules configured" : ""
                            color: Theme.surfaceText
                            font.pixelSize: Theme.fontSizeSmall
                            visible: root.rules.length === 0
                        }
                    }
                }

                StyledRect {
                    width: parent.width
                    height: categoriesColumn.implicitHeight + Theme.spacingL * 2
                    radius: Theme.cornerRadius
                    color: Theme.surfaceContainerHigh

                    Column {
                        id: categoriesColumn
                        anchors.fill: parent
                        anchors.margins: Theme.spacingL
                        spacing: Theme.spacingS

                        StyledText {
                            text: "Categories"
                            color: Theme.surfaceText
                            font.pixelSize: Theme.fontSizeLarge
                            font.bold: true
                        }

                        Repeater {
                            model: Object.keys(root.categories)

                            Row {
                                spacing: Theme.spacingS
                                StyledText {
                                    text: modelData
                                    color: Theme.surfaceText
                                    font.pixelSize: Theme.fontSizeSmall
                                }
                                StyledText {
                                    text: "(" + (root.categories[modelData]?.length || 0) + " patterns)"
                                    color: Theme.surfaceTextDim
                                    font.pixelSize: Theme.fontSizeSmall
                                }
                            }
                        }

                        StyledText {
                            text: Object.keys(root.categories).length === 0 ? "No categories configured" : ""
                            color: Theme.surfaceText
                            font.pixelSize: Theme.fontSizeSmall
                            visible: Object.keys(root.categories).length === 0
                        }
                    }
                }

                StyledRect {
                    width: parent.width
                    height: todosColumn.implicitHeight + Theme.spacingL * 2
                    radius: Theme.cornerRadius
                    color: Theme.surfaceContainerHigh

                    Column {
                        id: todosColumn
                        anchors.fill: parent
                        anchors.margins: Theme.spacingL
                        spacing: Theme.spacingS

                        StyledText {
                            text: "Todos (" + root.todos.length + ")"
                            color: Theme.surfaceText
                            font.pixelSize: Theme.fontSizeLarge
                            font.bold: true
                        }

                        Repeater {
                            model: root.todos

                            Row {
                                spacing: Theme.spacingS
                                DankIcon {
                                    name: modelData.done ? "check_box" : "check_box_outline_blank"
                                    color: modelData.done ? Theme.primary : Theme.surfaceText
                                    size: Theme.iconSizeSmall
                                    anchors.verticalCenter: parent.verticalCenter
                                }
                                StyledText {
                                    text: modelData.text || modelData.id || "Todo"
                                    color: modelData.done ? Theme.surfaceTextDim : Theme.surfaceText
                                    font.pixelSize: Theme.fontSizeSmall
                                    width: popout.width - Theme.spacingL * 2 - 50
                                    elide: Text.ElideRight
                                }
                            }
                        }

                        StyledText {
                            text: root.todos.length === 0 ? "No todos" : ""
                            color: Theme.surfaceText
                            font.pixelSize: Theme.fontSizeSmall
                            visible: root.todos.length === 0
                        }
                    }
                }
            }
        }
    }

    Component.onCompleted: {
        socket.connected = true
    }
}
