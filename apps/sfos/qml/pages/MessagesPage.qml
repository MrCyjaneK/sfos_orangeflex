import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All
    property var items: []

    function take() {
        if (flexClient.listName === "messages")
            page.items = flexClient.currentList
    }

    Component.onCompleted: flexClient.loadList("messages")

    Connections {
        target: flexClient
        onListChanged: page.take()
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        PullDownMenu {
            MenuItem {
                text: qsTr("Mark all read")
                enabled: !flexClient.busy
                onClicked: flexClient.markMessagesRead()
            }
            MenuItem {
                text: qsTr("Refresh")
                enabled: !flexClient.busy
                onClicked: flexClient.loadList("messages")
            }
        }

        Column {
            id: column
            width: parent.width

            PageHeader { title: qsTr("Notifications") }

            Label {
                visible: flexClient.error.length > 0 || flexClient.status.length > 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: flexClient.error.length ? flexClient.error : flexClient.status
            }

            Repeater {
                model: page.items
                ListItem {
                    id: messageItem
                    width: column.width
                    contentHeight: Math.max(Theme.itemSizeLarge, textColumn.height + 2 * Theme.paddingMedium)
                    Column {
                        id: textColumn
                        y: Theme.paddingMedium
                        x: Theme.horizontalPageMargin
                        width: parent.width - 2 * Theme.horizontalPageMargin
                        spacing: Theme.paddingSmall
                        Label {
                            width: parent.width
                            wrapMode: Text.Wrap
                            text: modelData.title
                            color: messageItem.highlighted
                                  ? Theme.highlightColor
                                  : (modelData.read === "true" ? Theme.secondaryColor : Theme.primaryColor)
                        }
                        Label {
                            width: parent.width
                            color: Theme.secondaryColor
                            font.pixelSize: Theme.fontSizeSmall
                            text: {
                                var bits = []
                                if (modelData.date)
                                    bits.push(modelData.date)
                                if (modelData.body)
                                    bits.push(modelData.body)
                                return bits.join(" · ")
                            }
                            truncationMode: TruncationMode.Fade
                        }
                    }
                    onClicked: pageStack.animatorPush(Qt.resolvedUrl("ArticlePage.qml"), {
                        "articleTitle": modelData.title,
                        "articleBody": modelData.body
                    })
                }
            }

            Label {
                visible: !flexClient.busy && page.items.length === 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("No messages.")
            }
        }

        VerticalScrollDecorator {}
    }
}
