import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All
    property var esims: []
    property var services: []

    function take() {
        if (flexClient.listName === "esims") {
            page.esims = flexClient.currentList
            if (!flexClient.busy)
                flexClient.loadList("services")
        } else if (flexClient.listName === "services") {
            page.services = flexClient.currentList
        }
    }

    Component.onCompleted: flexClient.loadList("esims")

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

            PageHeader {
                title: qsTr("My number")
                description: flexClient.offeringName
            }

            DetailItem {
                label: qsTr("Number")
                value: flexClient.msisdn
                visible: flexClient.msisdn.length > 0
            }

            DetailItem {
                label: qsTr("Status")
                value: flexClient.productStatus
                visible: flexClient.productStatus.length > 0
            }

            DetailItem {
                label: qsTr("Plan")
                value: flexClient.grantGB.length
                       ? qsTr("%1 GB").arg(flexClient.grantGB)
                       : flexClient.offeringName
            }

            DetailItem {
                label: qsTr("Left")
                value: flexClient.leftGB.length ? qsTr("%1 GB").arg(flexClient.leftGB) : ""
                visible: flexClient.leftGB.length > 0
            }

            DetailItem {
                label: qsTr("Renewal")
                value: flexClient.renewalTime.length
                       ? flexClient.renewalDate + " " + flexClient.renewalTime
                       : flexClient.renewalDate
                visible: flexClient.renewalDate.length > 0
            }

            Repeater {
                model: flexClient.flags
                DetailItem {
                    label: modelData.name
                    value: modelData.value
                }
            }

            SectionHeader {
                text: qsTr("SIM cards")
            }

            Label {
                visible: flexClient.multisimTotal > 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("MultiSIM %1 of %2 used").arg(flexClient.multisimUsed).arg(flexClient.multisimTotal)
            }

            Repeater {
                model: flexClient.sims
                ListItem {
                    contentHeight: Theme.itemSizeMedium
                    Label {
                        anchors.verticalCenter: parent.verticalCenter
                        x: Theme.horizontalPageMargin
                        width: parent.width - 2 * Theme.horizontalPageMargin
                        wrapMode: Text.Wrap
                        text: {
                            var bits = []
                            if (modelData.hierarchy)
                                bits.push(modelData.hierarchy)
                            if (modelData.type)
                                bits.push(modelData.type)
                            if (modelData.label)
                                bits.push(modelData.label)
                            return bits.join(" · ")
                        }
                    }
                }
            }

            Label {
                visible: flexClient.sims.length === 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("No SIM cards loaded.")
            }

            SectionHeader {
                visible: page.esims.length > 0
                text: qsTr("eSIM")
            }

            Repeater {
                model: page.esims
                Column {
                    width: column.width
                    DetailItem {
                        label: qsTr("State")
                        value: modelData.state + (modelData.subState ? " · " + modelData.subState : "")
                    }
                    DetailItem {
                        label: qsTr("Matching ID")
                        value: modelData.matchingId
                    }
                    DetailItem {
                        label: qsTr("ICCID")
                        value: modelData.iccid
                    }
                }
            }

            SectionHeader {
                visible: page.services.length > 0
                text: qsTr("Network services")
            }

            Repeater {
                model: page.services
                DetailItem {
                    label: modelData.title
                    value: modelData.value === "Y" ? qsTr("on") : (modelData.value === "N" ? qsTr("off") : modelData.value)
                }
            }
        }

        VerticalScrollDecorator {}
    }
}
