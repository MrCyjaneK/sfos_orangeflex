#ifdef QT_QML_DEBUG
#include <QtQuick>
#endif

#include <sailfishapp.h>
#include <QGuiApplication>
#include <QQuickView>
#include <QQmlContext>
#include <QtQml>

#include "clientbridge.h"

int main(int argc, char *argv[])
{
    QScopedPointer<QGuiApplication> app(SailfishApp::application(argc, argv));
    app->setOrganizationName(QStringLiteral("x.x.orangeflex"));
    app->setApplicationName(QStringLiteral("orangeflex"));

    QScopedPointer<QQuickView> view(SailfishApp::createView());
    ClientBridge bridge;
    view->rootContext()->setContextProperty(QStringLiteral("flexClient"), &bridge);
    view->setSource(SailfishApp::pathToMainQml());
    view->show();
    return app->exec();
}
