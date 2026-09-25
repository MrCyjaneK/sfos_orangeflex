import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All

    Connections {
        target: flexClient
        onLoggedInChanged: {
            if (!flexClient.loggedIn)
                pageStack.replace(Qt.resolvedUrl("LoginPage.qml"))
        }
    }

    SilicaFlickable {
        id: flickable
        anchors.fill: parent
        contentHeight: column.height

        PullDownMenu {
            MenuItem {
                text: qsTr("Notifications")
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("MessagesPage.qml"))
            }
            MenuItem {
                text: qsTr("My number")
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("MyNumberPage.qml"))
            }
            MenuItem {
                text: qsTr("Group")
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("GroupPage.qml"))
            }
            MenuItem {
                text: qsTr("Help")
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("HelpPage.qml"))
            }
            MenuItem {
                text: qsTr("Payments")
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("PaymentsPage.qml"))
            }
            MenuItem {
                text: qsTr("Orders")
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("OrdersPage.qml"))
            }
            MenuItem {
                text: qsTr("Refresh")
                onClicked: flexClient.refreshAccount()
            }
            MenuItem {
                text: qsTr("Settings")
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("SettingsPage.qml"))
            }
        }

        Column {
            id: column
            width: page.width

            PageHeader {
                title: flexClient.displayName.length
                       ? qsTr("Hello, %1!").arg(flexClient.displayName)
                       : qsTr("Home")
            }

            Column {
                width: parent.width
                visible: flexClient.busy || flexClient.error.length > 0 || flexClient.status.length > 0

                Label {
                    x: Theme.horizontalPageMargin
                    width: parent.width - 2 * Theme.horizontalPageMargin
                    wrapMode: Text.Wrap
                    color: Theme.highlightColor
                    text: flexClient.status
                    visible: flexClient.status.length > 0
                }

                ProgressBar {
                    width: parent.width
                    indeterminate: true
                    visible: flexClient.busy
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

            Label {
                visible: flexClient.renewalDate.length > 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: flexClient.renewalTime.length
                      ? qsTr("Your data will renew on: %1 at %2").arg(flexClient.renewalDate).arg(flexClient.renewalTime)
                      : qsTr("Your data will renew on: %1").arg(flexClient.renewalDate)
            }

            Item { width: 1; height: Theme.paddingLarge }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                font.pixelSize: Theme.fontSizeExtraLarge
                text: flexClient.leftGB.length
                      ? qsTr("Left %1 GB").arg(flexClient.leftGB)
                      : qsTr("No data counter")
            }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: flexClient.grantGB.length
                      ? qsTr("%1 GB in Plan").arg(flexClient.grantGB)
                      : ""
            }

            ProgressBar {
                width: parent.width
                minimumValue: 0
                maximumValue: 1
                value: {
                    var left = parseFloat(flexClient.leftGB)
                    var grant = parseFloat(flexClient.grantGB)
                    if (!grant || grant <= 0 || isNaN(left))
                        return 0
                    return Math.max(0, Math.min(1, left / grant))
                }
                visible: flexClient.grantGB.length > 0
            }

            BackgroundItem {
                width: parent.width
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("OffersPage.qml"))
                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    x: Theme.horizontalPageMargin
                    color: highlighted ? Theme.highlightColor : Theme.primaryColor
                    text: qsTr("Add data")
                }
            }

            BackgroundItem {
                width: parent.width
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("TransferPage.qml"))
                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    x: Theme.horizontalPageMargin
                    color: highlighted ? Theme.highlightColor : Theme.primaryColor
                    text: qsTr("Transfer data")
                }
            }

            Label {
                visible: flexClient.roamingGrantGB.length > 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("EU roaming FUP: %1 / %2 GB").arg(flexClient.roamingLeftGB).arg(flexClient.roamingGrantGB)
            }

            BackgroundItem {
                width: parent.width
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("RoamingPage.qml"))
                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    x: Theme.horizontalPageMargin
                    color: highlighted ? Theme.highlightColor : Theme.primaryColor
                    text: qsTr("Roaming and countries")
                }
            }

            Repeater {
                model: flexClient.meters
                Label {
                    x: Theme.horizontalPageMargin
                    width: parent.width - 2 * Theme.horizontalPageMargin
                    wrapMode: Text.Wrap
                    color: Theme.secondaryColor
                    font.pixelSize: Theme.fontSizeSmall
                    text: modelData.name + ": " + modelData.value + (modelData.unit ? " " + modelData.unit : "")
                }
            }

            SectionHeader {
                text: qsTr("Data Safe")
            }

            BackgroundItem {
                width: parent.width
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("DataSafePage.qml"))
                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    x: Theme.horizontalPageMargin
                    color: highlighted ? Theme.highlightColor : Theme.primaryColor
                    text: flexClient.dataSafeGB.length
                          ? qsTr("%1 GB stashed").arg(flexClient.dataSafeGB)
                          : qsTr("Data Safe")
                }
            }

            SectionHeader {
                text: qsTr("Flex funds")
            }

            BackgroundItem {
                width: parent.width
                onClicked: pageStack.animatorPush(Qt.resolvedUrl("PaymentsPage.qml"))
                Label {
                    anchors.verticalCenter: parent.verticalCenter
                    x: Theme.horizontalPageMargin
                    color: highlighted ? Theme.highlightColor : Theme.primaryColor
                    text: flexClient.walletAmount.length
                          ? flexClient.walletAmount
                          : qsTr("Top up with BLIK")
                }
            }

            Label {
                visible: flexClient.msisdn.length > 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                font.pixelSize: Theme.fontSizeSmall
                text: flexClient.msisdn
            }
        }

        VerticalScrollDecorator {}
    }
}
