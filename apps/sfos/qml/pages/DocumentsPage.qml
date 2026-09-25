import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All
    property var items: []

    function take() {
        if (flexClient.listName === "documents")
            page.items = flexClient.currentList
    }

    Component.onCompleted: flexClient.loadList("documents")

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

            PageHeader { title: qsTr("Documents") }

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
                    id: docItem
                    width: column.width
                    contentHeight: Math.max(Theme.itemSizeSmall, docLabel.height + 2 * Theme.paddingMedium)
                    onClicked: if (modelData.url) Qt.openUrlExternally(modelData.url)

                    Label {
                        id: docLabel
                        x: Theme.horizontalPageMargin
                        y: Theme.paddingMedium
                        width: parent.width - 2 * Theme.horizontalPageMargin
                        wrapMode: Text.Wrap
                        text: modelData.title
                        color: docItem.highlighted ? Theme.highlightColor : Theme.primaryColor
                    }
                }
            }
        }

        VerticalScrollDecorator {}
    }
}
