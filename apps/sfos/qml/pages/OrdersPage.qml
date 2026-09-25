import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All
    property var items: []

    function take() {
        if (flexClient.listName === "orders")
            page.items = flexClient.currentList
    }

    Component.onCompleted: flexClient.loadList("orders")

    Connections {
        target: flexClient
        onListChanged: page.take()
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width

            PageHeader { title: qsTr("Orders") }

            Label {
                visible: flexClient.error.length > 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: flexClient.error
            }

            Repeater {
                model: page.items
                ListItem {
                    contentHeight: Theme.itemSizeLarge
                    Column {
                        anchors.verticalCenter: parent.verticalCenter
                        x: Theme.horizontalPageMargin
                        width: parent.width - 2 * Theme.horizontalPageMargin
                        Label {
                            width: parent.width
                            wrapMode: Text.Wrap
                            text: modelData.name
                        }
                        Label {
                            width: parent.width
                            wrapMode: Text.Wrap
                            color: Theme.secondaryColor
                            font.pixelSize: Theme.fontSizeSmall
                            text: [modelData.status, modelData.date, modelData.delivery, modelData.id].filter(function(s) { return s && s.length }).join(" · ")
                        }
                    }
                }
            }

            Label {
                visible: !flexClient.busy && page.items.length === 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("No orders.")
            }
        }

        VerticalScrollDecorator {}
    }
}
