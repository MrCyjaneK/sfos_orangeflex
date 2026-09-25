import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    allowedOrientations: Orientation.All
    property var payments: []
    property var invoices: []

    function take() {
        if (flexClient.listName === "payments") {
            page.payments = flexClient.currentList
            if (!flexClient.busy)
                flexClient.loadList("invoices")
        } else if (flexClient.listName === "invoices") {
            page.invoices = flexClient.currentList
        }
    }

    Component.onCompleted: flexClient.loadList("payments")

    Connections {
        target: flexClient
        onListChanged: page.take()
        onPaymentSucceeded: {
            paidLabel.visible = true
            blikField.text = ""
            flexClient.loadList("payments")
        }
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: parent.width

            PageHeader { title: qsTr("Payments") }

            Label {
                visible: flexClient.error.length > 0 || flexClient.status.length > 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: flexClient.error.length ? flexClient.error : flexClient.status
            }

            SectionHeader { text: qsTr("Top up with BLIK") }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("Adds Flex funds. Open your bank app and confirm the BLIK code after you tap Top up.")
            }

            TextField {
                id: amountField
                width: parent.width
                label: qsTr("PLN")
                placeholderText: qsTr("10")
                inputMethodHints: Qt.ImhDigitsOnly
                text: "10"
            }

            TextField {
                id: blikField
                width: parent.width
                label: qsTr("BLIK code")
                placeholderText: qsTr("6 digits")
                inputMethodHints: Qt.ImhDigitsOnly
                echoMode: TextInput.PasswordEchoOnEdit
                validator: RegExpValidator { regExp: /[0-9]{0,6}/ }
            }

            Button {
                anchors.horizontalCenter: parent.horizontalCenter
                text: qsTr("Top up")
                enabled: amountField.text.length > 0 && blikField.text.length === 6 && !flexClient.busy
                onClicked: flexClient.topUpBlik(amountField.text, blikField.text)
            }

            Label {
                id: paidLabel
                visible: false
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.highlightColor
                text: qsTr("Top-up completed")
            }

            SectionHeader { text: qsTr("Methods") }

            Repeater {
                model: flexClient.paymentMethods
                DetailItem {
                    label: modelData.type
                    value: {
                        var bits = []
                        if (modelData.detail)
                            bits.push(modelData.detail)
                        if (modelData.active === "true")
                            bits.push(qsTr("active"))
                        return bits.join(" · ")
                    }
                }
            }

            Label {
                visible: flexClient.paymentMethods.length === 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("No saved methods.")
            }

            SectionHeader { text: qsTr("Wallet top-ups") }

            Repeater {
                model: page.payments
                ListItem {
                    contentHeight: Theme.itemSizeMedium
                    Column {
                        anchors.verticalCenter: parent.verticalCenter
                        x: Theme.horizontalPageMargin
                        width: parent.width - 2 * Theme.horizontalPageMargin
                        Label {
                            width: parent.width
                            text: modelData.name
                        }
                        Label {
                            width: parent.width
                            color: Theme.secondaryColor
                            font.pixelSize: Theme.fontSizeSmall
                            text: [modelData.amount, modelData.currency, modelData.status, modelData.method, modelData.date].filter(function(s) { return s && s.length }).join(" · ")
                        }
                    }
                }
            }

            Label {
                visible: !flexClient.busy && page.payments.length === 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("No wallet top-ups.")
            }

            SectionHeader { text: qsTr("Invoices") }

            Repeater {
                model: page.invoices
                DetailItem {
                    label: modelData.id
                    value: [modelData.state, modelData.amount, modelData.date].filter(function(s) { return s && s.length && s !== "<nil>" }).join(" · ")
                }
            }

            Label {
                visible: page.invoices.length === 0 && flexClient.listName === "invoices" && !flexClient.busy
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("No invoices.")
            }

        }

        VerticalScrollDecorator {}
    }
}
