import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All
    property string section: "offers"
    property string pageTitle: qsTr("Add data")
    property var items: []

    function take() {
        if (flexClient.listName === page.section)
            page.items = flexClient.currentList
    }

    Component.onCompleted: flexClient.loadList(page.section)

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

            PageHeader { title: page.pageTitle }

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
                            text: {
                                var bits = []
                                if (modelData.price)
                                    bits.push(modelData.price + " " + (modelData.currency || "PLN"))
                                if (modelData.period)
                                    bits.push(qsTr("%1 days").arg(modelData.period))
                                if (modelData.zone)
                                    bits.push(modelData.zone)
                                return bits.join(" · ")
                            }
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
                text: qsTr("No offers for this plan.")
            }
        }

        VerticalScrollDecorator {}
    }
}
