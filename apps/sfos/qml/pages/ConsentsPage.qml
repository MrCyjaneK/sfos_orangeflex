import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All
    property var items: []

    function take() {
        if (flexClient.listName === "consents")
            page.items = flexClient.currentList
    }

    Component.onCompleted: flexClient.loadList("consents")

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

            PageHeader { title: qsTr("Consents") }

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
                Column {
                    width: column.width
                    SectionHeader {
                        text: modelData.title
                    }
                    Label {
                        x: Theme.horizontalPageMargin
                        width: parent.width - 2 * Theme.horizontalPageMargin
                        wrapMode: Text.Wrap
                        color: Theme.secondaryHighlightColor
                        text: modelData.accepted === "true" ? qsTr("Accepted") : qsTr("Not accepted")
                    }
                    Label {
                        x: Theme.horizontalPageMargin
                        width: parent.width - 2 * Theme.horizontalPageMargin
                        wrapMode: Text.Wrap
                        text: modelData.text
                    }
                }
            }
        }

        VerticalScrollDecorator {}
    }
}
