import QtQuick
import Quickshell.Io
import qs.Common
import qs.Services
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

    DankSocket {
        id: socket
        path: "/tmp/makima.sock"

        onConnectionStateChanged: {
            root.isConnected = connected
            if (connected) {
                requestStatus()
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

    function addRule(rule) {
        _sendRequest("rule.add", rule)
    }

    function removeRule(id) {
        _sendRequest("rule.remove", {id: id})
    }

    function addCategory(name, patterns) {
        _sendRequest("category.add", {name: name, patterns: patterns})
    }

    function addTodo(text, parentId) {
        _sendRequest("todo.add", {text: text, parent: parentId})
    }

    function completeTodo(id) {
        _sendRequest("todo.done", {id: id})
    }

    Component.onCompleted: {
        socket.connected = true
    }
}
