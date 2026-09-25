import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All
    property string presetTo: ""

    Connections {
        target: flexClient
        onTransferSucceeded: sentLabel.visible = true
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width

            PageHeader { title: qsTr("Transfer data") }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("Send GB from this plan to another Flex number. Leave at least 1 GB for yourself.")
            }

            TextField {
                id: toField
                width: parent.width
                label: qsTr("Recipient number")
                placeholderText: qsTr("48XXXXXXXXX")
                inputMethodHints: Qt.ImhDialableCharactersOnly
                text: page.presetTo
            }

            TextField {
                id: valueField
                width: parent.width
                label: qsTr("GB")
                placeholderText: qsTr("1")
                inputMethodHints: Qt.ImhFormattedNumbersOnly
                text: "1"
            }

            TextField {
                id: messageField
                width: parent.width
                label: qsTr("Message")
                placeholderText: qsTr("Optional")
            }

            Button {
                anchors.horizontalCenter: parent.horizontalCenter
                text: qsTr("Send")
                enabled: toField.text.length > 0 && valueField.text.length > 0 && !flexClient.busy
                onClicked: {
                    sentLabel.visible = false
                    flexClient.transferData(toField.text, valueField.text, messageField.text)
                }
            }

            Label {
                id: sentLabel
                visible: false
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: qsTr("Data sent")
            }

            Label {
                visible: flexClient.error.length > 0 || flexClient.status.length > 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: flexClient.error.length ? flexClient.error : flexClient.status
            }
        }

        VerticalScrollDecorator {}
    }
}
