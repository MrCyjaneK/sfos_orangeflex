import QtQuick 2.0
import Sailfish.Silica 1.0

CoverBackground {
    Column {
        anchors.centerIn: parent
        width: parent.width - 2 * Theme.paddingLarge
        spacing: Theme.paddingSmall

        Label {
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.Wrap
            text: flexClient.loggedIn && flexClient.displayName.length
                  ? qsTr("Hello, %1!").arg(flexClient.displayName)
                  : qsTr("Orange Flex")
            color: Theme.primaryColor
            font.pixelSize: Theme.fontSizeLarge
        }

        Label {
            visible: flexClient.loggedIn && flexClient.leftGB.length > 0
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.Wrap
            color: Theme.highlightColor
            text: qsTr("%1 / %2 GB").arg(flexClient.leftGB).arg(flexClient.grantGB)
        }
    }
}
