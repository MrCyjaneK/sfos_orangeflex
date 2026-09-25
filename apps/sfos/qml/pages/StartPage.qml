import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All

    property bool loginOpened: false

    function maybeNavigate() {
        if (!flexClient.ready)
            return
        if (flexClient.loggedIn) {
            pageStack.replace(Qt.resolvedUrl("HomePage.qml"))
            return
        }
        if (page.loginOpened)
            return
        page.loginOpened = true
        pageStack.replace(Qt.resolvedUrl("LoginPage.qml"))
    }

    Component.onCompleted: flexClient.startInit()

    Connections {
        target: flexClient
        onReadyChanged: page.maybeNavigate()
        onLoggedInChanged: page.maybeNavigate()
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: page.width

            PageHeader {
                title: qsTr("Orange Flex")
            }

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
    }
}
