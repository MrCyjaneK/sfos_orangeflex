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
                title: qsTr("Group")
                description: flexClient.groupType
            }

            Label {
                visible: flexClient.members.length === 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("No group members.")
            }

            Repeater {
                model: flexClient.members
                ListItem {
                    contentHeight: Theme.itemSizeLarge
                    onClicked: pageStack.animatorPush(Qt.resolvedUrl("TransferPage.qml"), { presetTo: modelData.msisdn })
                    Column {
                        anchors.verticalCenter: parent.verticalCenter
                        x: Theme.horizontalPageMargin
                        width: parent.width - 2 * Theme.horizontalPageMargin

                        Label {
                            width: parent.width
                            wrapMode: Text.Wrap
                            text: modelData.alias.length ? modelData.alias : modelData.msisdn
                        }

                        Label {
                            width: parent.width
                            wrapMode: Text.Wrap
                            color: Theme.secondaryColor
                            font.pixelSize: Theme.fontSizeSmall
                            text: {
                                var bits = []
                                if (modelData.role)
                                    bits.push(modelData.role)
                                if (modelData.status)
                                    bits.push(modelData.status)
                                if (modelData.leftGB && modelData.grantGB)
                                    bits.push(qsTr("%1 / %2 GB").arg(modelData.leftGB).arg(modelData.grantGB))
                                else if (modelData.msisdn)
                                    bits.push(modelData.msisdn)
                                return bits.join(" · ")
                            }
                        }
                    }
                }
            }
        }

        VerticalScrollDecorator {}
    }
}
