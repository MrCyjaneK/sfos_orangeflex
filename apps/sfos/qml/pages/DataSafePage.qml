import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All

    Connections {
        target: flexClient
        onWithdrawSucceeded: withdrawnLabel.visible = true
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width

            PageHeader {
                title: qsTr("Data Safe")
                description: flexClient.dataSafeGB.length
                             ? qsTr("%1 GB stashed").arg(flexClient.dataSafeGB)
                             : ""
            }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("Unused GB from earlier months, each period expiring after six months. Withdrawn data can be used until the end of this billing period.")
            }

            TextField {
                id: amountField
                width: parent.width
                label: qsTr("GB to withdraw")
                placeholderText: qsTr("1")
                inputMethodHints: Qt.ImhFormattedNumbersOnly
                text: "1"
            }

            Button {
                anchors.horizontalCenter: parent.horizontalCenter
                text: qsTr("Withdraw")
                enabled: amountField.text.length > 0 && !flexClient.busy && flexClient.dataSafeGB.length > 0
                onClicked: {
                    withdrawnLabel.visible = false
                    remorse.execute(qsTr("Withdraw %1 GB").arg(amountField.text), function() {
                        flexClient.withdrawDataSafe(amountField.text)
                    })
                }
            }

            Label {
                id: withdrawnLabel
                visible: false
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: qsTr("Withdrawn from the Safe")
            }

            Label {
                visible: flexClient.error.length > 0 || flexClient.status.length > 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: flexClient.error.length ? flexClient.error : flexClient.status
            }

            Repeater {
                model: flexClient.dataSafePeriods
                ListItem {
                    contentHeight: Theme.itemSizeMedium
                    onClicked: amountField.text = modelData.amount
                    Column {
                        anchors.verticalCenter: parent.verticalCenter
                        x: Theme.horizontalPageMargin
                        width: parent.width - 2 * Theme.horizontalPageMargin
                        Label {
                            text: qsTr("%1 GB").arg(modelData.amount)
                        }
                        Label {
                            color: Theme.secondaryColor
                            font.pixelSize: Theme.fontSizeSmall
                            text: {
                                var bits = []
                                if (modelData.activation)
                                    bits.push(qsTr("from %1").arg(modelData.activation))
                                if (modelData.expiry)
                                    bits.push(qsTr("until %1").arg(modelData.expiry))
                                return bits.join(" · ")
                            }
                        }
                    }
                }
            }

            Label {
                visible: flexClient.dataSafePeriods.length === 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("No stash periods.")
            }
        }

        RemorsePopup { id: remorse }

        VerticalScrollDecorator {}
    }
}
