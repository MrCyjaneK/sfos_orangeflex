import QtQuick 2.0
import Sailfish.Silica 1.0

Page {
    id: page
    objectName: "loginPage"
    allowedOrientations: Orientation.All

    // 0 = number, 1 = SMS code, 2 = email code
    property int step: 0

    function resetForm() {
        page.step = 0
        otpField.text = ""
    }

    Connections {
        target: flexClient
        onLoggedInChanged: {
            if (flexClient.loggedIn)
                pageStack.replace(Qt.resolvedUrl("HomePage.qml"))
        }
        onOtpSent: {
            page.step = 1
            otpField.text = ""
        }
        onNeedNextOtp: {
            page.step = 2
            otpField.text = ""
        }
    }

    SilicaFlickable {
        anchors.fill: parent
        contentHeight: column.height

        Column {
            id: column
            width: page.width

            PageHeader {
                title: qsTr("Sign in")
            }

            Label {
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                wrapMode: Text.Wrap
                color: Theme.secondaryColor
                text: qsTr("Enter your Flex number. A one-time code will be sent by SMS. If Orange asks for email confirmation, a second code follows. An existing account on a new device may then send another SMS.")
            }

            TextField {
                id: identityField
                width: parent.width
                label: qsTr("Flex number")
                placeholderText: qsTr("48XXXXXXXXX")
                inputMethodHints: Qt.ImhDialableCharactersOnly
                EnterKey.enabled: text.length > 0 && !flexClient.busy
                EnterKey.onClicked: flexClient.loginStart(text)
                enabled: page.step === 0
            }

            TextField {
                id: otpField
                width: parent.width
                visible: page.step > 0
                label: page.step === 2
                       ? (flexClient.pendingEmailMask.length
                          ? qsTr("Email code sent to %1").arg(flexClient.pendingEmailMask)
                          : qsTr("Email code"))
                       : qsTr("SMS code")
                placeholderText: qsTr("000000")
                inputMethodHints: Qt.ImhDigitsOnly
                EnterKey.enabled: text.length > 0 && !flexClient.busy
                EnterKey.onClicked: flexClient.loginSubmitOtp(text)
            }

            Label {
                visible: page.step > 0 && flexClient.loginTimeout > 0
                x: Theme.horizontalPageMargin
                width: parent.width - 2 * Theme.horizontalPageMargin
                color: Theme.secondaryColor
                font.pixelSize: Theme.fontSizeSmall
                text: qsTr("Code is valid for %1 seconds").arg(flexClient.loginTimeout)
            }

            Column {
                width: parent.width
                spacing: Theme.paddingMedium

                Button {
                    anchors.horizontalCenter: parent.horizontalCenter
                    visible: page.step === 0
                    text: qsTr("Send code")
                    enabled: identityField.text.length > 0 && !flexClient.busy
                    onClicked: flexClient.loginStart(identityField.text)
                }

                Button {
                    anchors.horizontalCenter: parent.horizontalCenter
                    visible: page.step > 0
                    text: qsTr("Confirm")
                    enabled: otpField.text.length > 0 && !flexClient.busy
                    onClicked: flexClient.loginSubmitOtp(otpField.text)
                }

                Button {
                    anchors.horizontalCenter: parent.horizontalCenter
                    visible: page.step > 0
                    text: qsTr("Change number")
                    enabled: !flexClient.busy
                    onClicked: page.resetForm()
                }
            }

            Item { width: 1; height: Theme.paddingLarge }

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
