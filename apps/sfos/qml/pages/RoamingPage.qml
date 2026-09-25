import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All
    property var allCountries: []
    property string query: ""

    ListModel { id: countryModel }

    function take() {
        if (flexClient.listName !== "countries")
            return
        page.allCountries = flexClient.currentList
        page.applyFilter()
    }

    function applyFilter() {
        var src = page.allCountries || []
        var q = page.query.toLowerCase()
        countryModel.clear()
        for (var i = 0; i < src.length; i++) {
            var row = src[i]
            var hay = ((row.name || "") + " " + (row.iso || "") + " " + (row.zone || "")).toLowerCase()
            if (q.length && hay.indexOf(q) === -1)
                continue
            countryModel.append({
                "name": row.name || "",
                "iso": row.iso || "",
                "zone": row.zone || "",
                "description": row.description || ""
            })
        }
    }

    function openCountry(name, iso, zone, description) {
        var bits = []
        if (iso)
            bits.push(iso)
        if (zone)
            bits.push(zone)
        if (description)
            bits.push(description)
        pageStack.animatorPush(Qt.resolvedUrl("ArticlePage.qml"), {
            "articleTitle": name,
            "articleBody": bits.join("\n\n")
        })
    }

    Component.onCompleted: flexClient.loadList("countries")

    Connections {
        target: flexClient
        onListChanged: page.take()
    }

    Column {
        id: headerCol
        width: parent.width

        PageHeader { title: qsTr("Roaming") }

        Label {
            visible: flexClient.roamingGrantGB.length > 0
            x: Theme.horizontalPageMargin
            width: parent.width - 2 * Theme.horizontalPageMargin
            wrapMode: Text.Wrap
            text: qsTr("EU roaming FUP: %1 / %2 GB").arg(flexClient.roamingLeftGB).arg(flexClient.roamingGrantGB)
        }

        BackgroundItem {
            width: parent.width
            onClicked: pageStack.animatorPush(Qt.resolvedUrl("OffersPage.qml"), {
                "section": "roamingOffers",
                "pageTitle": qsTr("Roaming packs")
            })
            Label {
                anchors.verticalCenter: parent.verticalCenter
                x: Theme.horizontalPageMargin
                color: highlighted ? Theme.highlightColor : Theme.primaryColor
                text: qsTr("Roaming packs")
            }
        }

        SearchField {
            id: searchField
            width: parent.width
            placeholderText: qsTr("Country")
            onTextChanged: {
                page.query = text
                page.applyFilter()
            }
        }

        Label {
            visible: flexClient.error.length > 0
            x: Theme.horizontalPageMargin
            width: parent.width - 2 * Theme.horizontalPageMargin
            wrapMode: Text.Wrap
            color: Theme.highlightColor
            text: flexClient.error
        }
    }

    SilicaListView {
        id: listView
        anchors.top: headerCol.bottom
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        clip: true
        model: countryModel

        delegate: ListItem {
            id: countryItem
            width: listView.width
            contentHeight: Theme.itemSizeSmall
            onClicked: page.openCountry(name, iso, zone, description)

            Label {
                anchors.verticalCenter: parent.verticalCenter
                x: Theme.horizontalPageMargin
                width: parent.width - zoneLabel.width - 3 * Theme.horizontalPageMargin
                truncationMode: TruncationMode.Fade
                text: name + (iso ? " (" + iso + ")" : "")
                color: countryItem.highlighted ? Theme.highlightColor : Theme.primaryColor
            }

            Label {
                id: zoneLabel
                anchors.verticalCenter: parent.verticalCenter
                anchors.right: parent.right
                anchors.rightMargin: Theme.horizontalPageMargin
                color: Theme.secondaryColor
                font.pixelSize: Theme.fontSizeSmall
                text: zone
            }
        }

        ViewPlaceholder {
            enabled: !flexClient.busy && countryModel.count === 0
            text: qsTr("No countries")
        }

        VerticalScrollDecorator {}
    }
}
