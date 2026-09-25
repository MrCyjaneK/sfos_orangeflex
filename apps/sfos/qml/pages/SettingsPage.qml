import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    allowedOrientations: Orientation.All

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width

            PageHeader {
                title: qsTr("Settings")
            }

            SectionHeader {
                text: qsTr("Language")
            }

            ComboBox {
                id: langBox
                width: parent.width
                label: qsTr("API language")
                currentIndex: {
                    if (flexClient.language === "pl")
                        return 1
                    if (flexClient.language === "uk")
                        return 2
                    return 0
                }
                menu: ContextMenu {
                    MenuItem { text: qsTr("English") }
                    MenuItem { text: qsTr("Polish") }
                    MenuItem { text: qsTr("Ukrainian") }
                }
                onCurrentIndexChanged: {
                    var langs = ["en", "pl", "uk"]
                    flexClient.setLanguage(langs[currentIndex])
                }
            }

            SectionHeader {
                text: qsTr("Account")
            }

            DetailItem {
                label: qsTr("Name")
                value: (flexClient.firstName + " " + flexClient.lastName).trim()
                visible: flexClient.firstName.length > 0 || flexClient.lastName.length > 0
            }

            DetailItem {
                label: qsTr("Email")
                value: flexClient.email
                visible: flexClient.email.length > 0
            }

            DetailItem {
                label: qsTr("Number")
                value: flexClient.msisdn
                visible: flexClient.msisdn.length > 0
            }

            BackgroundItem {
                width: parent.width
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("PaymentsPage.qml"))
                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    x: Theme.horizontalPageMargin
                    color: highlighted ? Theme.highlightColor : Theme.primaryColor
                    text: qsTr("Payments")
                }
            }

            BackgroundItem {
                width: parent.width
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("OrdersPage.qml"))
                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    x: Theme.horizontalPageMargin
                    color: highlighted ? Theme.highlightColor : Theme.primaryColor
                    text: qsTr("Orders")
                }
            }

            BackgroundItem {
                width: parent.width
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("ConsentsPage.qml"))
                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    x: Theme.horizontalPageMargin
                    color: highlighted ? Theme.highlightColor : Theme.primaryColor
                    text: qsTr("Consents")
                }
            }

            BackgroundItem {
                width: parent.width
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("DocumentsPage.qml"))
                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    x: Theme.horizontalPageMargin
                    color: highlighted ? Theme.highlightColor : Theme.primaryColor
                    text: qsTr("Documents")
                }
            }

            BackgroundItem {
                width: parent.width
                onClicked: flexClient.logout()
                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    x: Theme.horizontalPageMargin
                    color: highlighted ? Theme.highlightColor : Theme.primaryColor
                    text: qsTr("Sign out")
                }
            }

            SectionHeader {
                text: qsTr("About")
            }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                text: qsTr("Unofficial Sailfish OS client.")
                color: Theme.secondaryColor
                font.pixelSize: Theme.fontSizeSmall
            }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                font.pixelSize: Theme.fontSizeSmall
                text: qsTr("Version 0.2")
            }

            SectionHeader {
                text: qsTr("Data")
            }

            BackgroundItem {
                width: parent.width
                onClicked: remorse.execute(qsTr("Deleting all data and closing"), function() {
                    flexClient.wipeAndExit()
                })
                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    x: Theme.horizontalPageMargin
                    color: highlighted ? Theme.highlightColor : Theme.primaryColor
                    text: qsTr("Delete all data and exit")
                }
            }
        }

        RemorsePopup { id: remorse }

        VerticalScrollDecorator {}
    }
}
