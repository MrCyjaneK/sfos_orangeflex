import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All
    property var items: []

    function take() {
        if (flexClient.listName !== "faq")
            return
        var cats = flexClient.currentList
        var rows = []
        for (var i = 0; i < cats.length; i++) {
            var cat = cats[i]
            rows.push({ kind: "section", title: cat.title || "", content: "" })
            var arts = cat.articles || []
            for (var j = 0; j < arts.length; j++) {
                rows.push({
                    kind: "article",
                    title: arts[j].title || "",
                    content: arts[j].content || ""
                })
            }
        }
        page.items = rows
    }

    Component.onCompleted: flexClient.loadList("faq")

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

            PageHeader { title: qsTr("Help") }

            Label {
                visible: flexClient.busy || flexClient.error.length > 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: flexClient.error.length ? flexClient.error : flexClient.status
            }

            Repeater {
                model: page.items
                Item {
                    width: column.width
                    height: childrenRect.height

                    SectionHeader {
                        width: parent.width
                        visible: modelData.kind === "section"
                        height: visible ? implicitHeight : 0
                        text: modelData.title
                    }

                    ListItem {
                        id: articleItem
                        width: parent.width
                        visible: modelData.kind === "article"
                        contentHeight: Math.max(Theme.itemSizeSmall, articleLabel.height + 2 * Theme.paddingMedium)
                        height: visible ? contentHeight : 0
                        onClicked: pageStack.animatorPush(Qt.resolvedUrl("ArticlePage.qml"), {
                            "articleTitle": modelData.title,
                            "articleBody": modelData.content
                        })

                        Label {
                            id: articleLabel
                            x: Theme.horizontalPageMargin
                            y: Theme.paddingMedium
                            width: parent.width - 2 * Theme.horizontalPageMargin
                            wrapMode: Text.Wrap
                            text: modelData.title
                            color: articleItem.highlighted ? Theme.highlightColor : Theme.primaryColor
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
                text: qsTr("No articles.")
            }
        }

        VerticalScrollDecorator {}
    }
}
