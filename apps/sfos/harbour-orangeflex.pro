# NOTICE:
#
# Application name defined in TARGET has a corresponding QML filename.
# If name defined in TARGET is changed, the following needs to be done
# to match new name:
#   - corresponding QML filename must be changed
#   - desktop icon filename must be changed
#   - desktop filename must be changed
#   - icon definition filename in desktop file must be changed

TARGET = harbour-orangeflex

CONFIG += sailfishapp c++11

INCLUDEPATH += $$PWD/go $$PWD/src

LIBS += -L$$PWD/lib -lorangeflex
QMAKE_LFLAGS += -Wl,-rpath,/usr/share/harbour-orangeflex/lib

CAPI_LIB = $$PWD/lib/liborangeflex.so
capi.target = $$CAPI_LIB
capi.commands = sh $$PWD/go/build.sh
capi.depends = $$PWD/go/main.go \
    $$PWD/go/go.mod \
    $$files($$PWD/go/client/*.go)
QMAKE_EXTRA_TARGETS += capi
PRE_TARGETDEPS += $$CAPI_LIB

HEADERS += src/clientbridge.h \
    go/orangeflex.h

SOURCES += src/harbour-orangeflex.cpp \
    src/clientbridge.cpp

lib.files = lib/liborangeflex.so
lib.path = /usr/share/$${TARGET}/lib
INSTALLS += lib

DISTFILES += qml/harbour-orangeflex.qml \
    qml/cover/CoverPage.qml \
    qml/pages/StartPage.qml \
    qml/pages/LoginPage.qml \
    qml/pages/HomePage.qml \
    qml/pages/MyNumberPage.qml \
    qml/pages/GroupPage.qml \
    qml/pages/SettingsPage.qml \
    qml/pages/HelpPage.qml \
    qml/pages/ArticlePage.qml \
    qml/pages/MessagesPage.qml \
    qml/pages/PaymentsPage.qml \
    qml/pages/OrdersPage.qml \
    qml/pages/OffersPage.qml \
    qml/pages/RoamingPage.qml \
    qml/pages/TransferPage.qml \
    qml/pages/DataSafePage.qml \
    qml/pages/ConsentsPage.qml \
    qml/pages/DocumentsPage.qml \
    rpm/harbour-orangeflex.spec \
    harbour-orangeflex.desktop

SAILFISHAPP_ICONS = 86x86 108x108 128x128 172x172
