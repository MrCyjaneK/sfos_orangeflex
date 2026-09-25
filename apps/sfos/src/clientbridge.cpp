#include "clientbridge.h"
#include "orangeflex.h"

#include <QCoreApplication>
#include <QSettings>
#include <QStandardPaths>
#include <QDir>
#include <QMetaType>
#include <QJsonDocument>
#include <QJsonArray>
#include <QJsonObject>
#include <string>
#include <thread>

static ClientBridge *g_bridge = 0;

static void orangeflex_progress_c(const char *step, const char *message, int64_t bytes, int64_t total)
{
    if (!g_bridge)
        return;
    QMetaObject::invokeMethod(g_bridge, "onProgress", Qt::QueuedConnection,
                              Q_ARG(QString, QString::fromUtf8(step ? step : "")),
                              Q_ARG(QString, QString::fromUtf8(message ? message : "")),
                              Q_ARG(qint64, (qint64)bytes),
                              Q_ARG(qint64, (qint64)total));
}

ClientBridge::ClientBridge(QObject *parent)
    : QObject(parent)
    , m_networkAllowed(true)
    , m_needsNetworkPrompt(false)
    , m_busy(false)
    , m_ready(false)
    , m_loggedIn(false)
    , m_status(QString())
    , m_progress(-1.0)
    , m_loginTimeout(0)
    , m_language(QStringLiteral("en"))
    , m_multisimUsed(0)
    , m_multisimTotal(0)
    , m_multisimAvailable(0)
{
    g_bridge = this;
    qRegisterMetaType<qint64>("qint64");
    OrangeFlex_SetProgressHandler(orangeflex_progress_c);

    QString cache = QStandardPaths::writableLocation(QStandardPaths::CacheLocation);
    QDir().mkpath(cache);
    QByteArray cacheUtf8 = cache.toUtf8();
    OrangeFlex_SetCacheDir(cacheUtf8.constData());

    OrangeFlex_SetNetworkAllowed(1);
}

QString ClientBridge::takeString(char *p) const
{
    QString s = QString::fromUtf8(p ? p : "");
    OrangeFlex_Free(p);
    return s;
}

void ClientBridge::beginBusy(const QString &status)
{
    m_busy = true;
    m_error.clear();
    m_progress = -1.0;
    m_status = status;
    emit busyChanged();
    emit errorChanged();
    emit progressChanged();
    emit statusChanged();
}

void ClientBridge::applyLoggedIn(bool value)
{
    if (m_loggedIn == value)
        return;
    m_loggedIn = value;
    emit loggedInChanged();
}

void ClientBridge::refreshLoginFields()
{
    m_loginTimeout = OrangeFlex_LoginTimeout();
    m_pendingEmailMask = takeString(OrangeFlex_PendingEmailMask());
    m_pendingMsisdn = takeString(OrangeFlex_PendingMSISDN());
    emit loginChanged();
}

void ClientBridge::refreshAccountFields()
{
    m_language = takeString(OrangeFlex_Language());
    m_displayName = takeString(OrangeFlex_DisplayName());
    m_firstName = takeString(OrangeFlex_FirstName());
    m_lastName = takeString(OrangeFlex_LastName());
    m_email = takeString(OrangeFlex_Email());
    m_msisdn = takeString(OrangeFlex_MSISDN());
    m_offeringName = takeString(OrangeFlex_OfferingName());
    m_productStatus = takeString(OrangeFlex_ProductStatus());
    m_renewalDate = takeString(OrangeFlex_RenewalDate());
    m_renewalTime = takeString(OrangeFlex_RenewalTime());
    m_leftGB = takeString(OrangeFlex_LeftGB());
    m_grantGB = takeString(OrangeFlex_GrantGB());
    m_walletAmount = takeString(OrangeFlex_WalletAmount());
    m_dataSafeGB = takeString(OrangeFlex_DataSafeGB());
    m_groupType = takeString(OrangeFlex_GroupType());
    m_multisimUsed = OrangeFlex_MultisimUsed();
    m_multisimTotal = OrangeFlex_MultisimTotal();
    m_multisimAvailable = OrangeFlex_MultisimAvailable();

    QVariantList sims;
    const int simCount = OrangeFlex_SIMCount();
    for (int i = 0; i < simCount; ++i) {
        QVariantMap row;
        row.insert(QStringLiteral("hierarchy"), takeString(OrangeFlex_SIMHierarchy(i)));
        row.insert(QStringLiteral("type"), takeString(OrangeFlex_SIMType(i)));
        row.insert(QStringLiteral("label"), takeString(OrangeFlex_SIMLabel(i)));
        sims.append(row);
    }
    m_sims = sims;

    QVariantList members;
    const int memberCount = OrangeFlex_MemberCount();
    for (int i = 0; i < memberCount; ++i) {
        QVariantMap row;
        row.insert(QStringLiteral("alias"), takeString(OrangeFlex_MemberAlias(i)));
        row.insert(QStringLiteral("role"), takeString(OrangeFlex_MemberRole(i)));
        row.insert(QStringLiteral("msisdn"), takeString(OrangeFlex_MemberMSISDN(i)));
        row.insert(QStringLiteral("status"), takeString(OrangeFlex_MemberStatus(i)));
        row.insert(QStringLiteral("leftGB"), takeString(OrangeFlex_MemberLeftGB(i)));
        row.insert(QStringLiteral("grantGB"), takeString(OrangeFlex_MemberGrantGB(i)));
        members.append(row);
    }
    m_members = members;

    applyExtraJson();

    emit languageChanged();
    emit dataChanged();
}

void ClientBridge::applyExtraJson()
{
    const QByteArray raw = takeString(OrangeFlex_ExtraJSON()).toUtf8();
    const QJsonDocument doc = QJsonDocument::fromJson(raw);
    const QJsonObject o = doc.object();
    m_roamingLeftGB = o.value(QStringLiteral("roamingLeft")).toString();
    m_roamingGrantGB = o.value(QStringLiteral("roamingGrant")).toString();
    m_dataSafePeriods = o.value(QStringLiteral("periods")).toArray().toVariantList();
    m_meters = o.value(QStringLiteral("meters")).toArray().toVariantList();
    m_paymentMethods = o.value(QStringLiteral("methods")).toArray().toVariantList();
    m_flags = o.value(QStringLiteral("flags")).toArray().toVariantList();
}

void ClientBridge::applyListJson()
{
    m_listName = takeString(OrangeFlex_ListName());
    const QByteArray raw = takeString(OrangeFlex_ListJSON()).toUtf8();
    const QJsonDocument doc = QJsonDocument::fromJson(raw);
    if (doc.isArray())
        m_currentList = doc.array().toVariantList();
    else
        m_currentList = QVariantList();
    emit listChanged();
}

void ClientBridge::finishAction(int rc, const QString &fallback)
{
    m_busy = false;
    emit busyChanged();
    if (rc != 0) {
        m_error = takeString(OrangeFlex_LastError());
        if (m_error.isEmpty())
            m_error = fallback;
        m_status = tr("Failed");
        emit errorChanged();
        emit statusChanged();
        return;
    }
    m_error.clear();
    m_progress = -1.0;
    m_status.clear();
    emit errorChanged();
    emit progressChanged();
    emit statusChanged();
}

void ClientBridge::setNetworkAllowed(bool allow)
{
    m_networkAllowed = allow;
    m_needsNetworkPrompt = false;
    OrangeFlex_SetNetworkAllowed(allow ? 1 : 0);
    QSettings settings;
    settings.setValue(QStringLiteral("networkAllowed"), allow);
    settings.sync();
    emit networkAllowedChanged();
    emit needsNetworkPromptChanged();
    if (!allow) {
        m_status = tr("Network permission declined");
        emit statusChanged();
    }
}

void ClientBridge::startInit()
{
    if (m_busy)
        return;
    m_ready = false;
    emit readyChanged();
    beginBusy(tr("Starting…"));
    std::thread([this]() {
        int rc = OrangeFlex_Init();
        QMetaObject::invokeMethod(this, "initFinished", Qt::QueuedConnection, Q_ARG(int, rc));
    }).detach();
}

void ClientBridge::loginStart(const QString &to)
{
    if (m_busy)
        return;
    beginBusy(tr("Sending one-time code"));
    const std::string utf8 = to.toUtf8().constData();
    std::thread([this, utf8]() {
        int rc = OrangeFlex_LoginStart(utf8.c_str());
        QMetaObject::invokeMethod(this, "loginStartFinished", Qt::QueuedConnection, Q_ARG(int, rc));
    }).detach();
}

void ClientBridge::loginSubmitOtp(const QString &otp)
{
    if (m_busy)
        return;
    beginBusy(tr("Checking one-time code"));
    const std::string utf8 = otp.toUtf8().constData();
    std::thread([this, utf8]() {
        int rc = OrangeFlex_LoginSubmitOTP(utf8.c_str());
        QMetaObject::invokeMethod(this, "loginOtpFinished", Qt::QueuedConnection, Q_ARG(int, rc));
    }).detach();
}

void ClientBridge::refreshAccount()
{
    if (m_busy)
        return;
    beginBusy(tr("Refreshing"));
    std::thread([this]() {
        int rc = OrangeFlex_Refresh();
        QMetaObject::invokeMethod(this, "refreshFinished", Qt::QueuedConnection, Q_ARG(int, rc));
    }).detach();
}

void ClientBridge::logout()
{
    if (m_busy)
        return;
    beginBusy(tr("Signing out"));
    std::thread([this]() {
        int rc = OrangeFlex_Logout();
        QMetaObject::invokeMethod(this, "logoutFinished", Qt::QueuedConnection, Q_ARG(int, rc));
    }).detach();
}

void ClientBridge::setLanguage(const QString &lang)
{
    QByteArray utf8 = lang.toUtf8();
    OrangeFlex_SetLanguage(utf8.constData());
    m_language = takeString(OrangeFlex_Language());
    emit languageChanged();
}

void ClientBridge::wipeAndExit()
{
    if (m_busy)
        return;
    beginBusy(tr("Deleting all data…"));
    std::thread([this]() {
        int rc = OrangeFlex_WipeData();
        QMetaObject::invokeMethod(this, "wipeFinished", Qt::QueuedConnection, Q_ARG(int, rc));
    }).detach();
}

void ClientBridge::loadList(const QString &name)
{
    if (m_busy)
        return;
    beginBusy(tr("Loading"));
    const std::string utf8 = name.toUtf8().constData();
    std::thread([this, utf8]() {
        int rc = OrangeFlex_LoadList(utf8.c_str());
        QMetaObject::invokeMethod(this, "loadListFinished", Qt::QueuedConnection, Q_ARG(int, rc));
    }).detach();
}

void ClientBridge::transferData(const QString &to, const QString &value, const QString &message)
{
    if (m_busy)
        return;
    beginBusy(tr("Sending data"));
    const std::string toUtf8 = to.toUtf8().constData();
    const std::string valueUtf8 = value.toUtf8().constData();
    const std::string msgUtf8 = message.toUtf8().constData();
    std::thread([this, toUtf8, valueUtf8, msgUtf8]() {
        int rc = OrangeFlex_TransferData(toUtf8.c_str(), valueUtf8.c_str(), msgUtf8.c_str());
        QMetaObject::invokeMethod(this, "transferFinished", Qt::QueuedConnection, Q_ARG(int, rc));
    }).detach();
}

void ClientBridge::topUpBlik(const QString &amount, const QString &authCode)
{
    if (m_busy)
        return;
    beginBusy(tr("Paying with BLIK"));
    const std::string amountUtf8 = amount.toUtf8().constData();
    const std::string codeUtf8 = authCode.toUtf8().constData();
    std::thread([this, amountUtf8, codeUtf8]() {
        int rc = OrangeFlex_TopUpBlik(amountUtf8.c_str(), codeUtf8.c_str());
        QMetaObject::invokeMethod(this, "topUpFinished", Qt::QueuedConnection, Q_ARG(int, rc));
    }).detach();
}

void ClientBridge::withdrawDataSafe(const QString &amount)
{
    if (m_busy)
        return;
    beginBusy(tr("Withdrawing data"));
    const std::string amountUtf8 = amount.toUtf8().constData();
    std::thread([this, amountUtf8]() {
        int rc = OrangeFlex_WithdrawDataSafe(amountUtf8.c_str());
        QMetaObject::invokeMethod(this, "withdrawFinished", Qt::QueuedConnection, Q_ARG(int, rc));
    }).detach();
}

void ClientBridge::markMessagesRead()
{
    if (m_busy)
        return;
    beginBusy(tr("Marking read"));
    std::thread([this]() {
        int rc = OrangeFlex_MarkMessagesRead();
        QMetaObject::invokeMethod(this, "markReadFinished", Qt::QueuedConnection, Q_ARG(int, rc));
    }).detach();
}

void ClientBridge::onProgress(const QString &step, const QString &message, qint64 bytes, qint64 total)
{
    Q_UNUSED(bytes);
    Q_UNUSED(total);
    if (step.isEmpty())
        m_status = message;
    else if (message.isEmpty())
        m_status = step;
    else
        m_status = step + QStringLiteral(": ") + message;
    m_progress = -1.0;
    emit statusChanged();
    emit progressChanged();
}

void ClientBridge::initFinished(int rc)
{
    m_busy = false;
    emit busyChanged();
    if (rc != 0) {
        m_error = takeString(OrangeFlex_LastError());
        if (m_error.isEmpty())
            m_error = tr("Init failed");
        m_status = tr("Failed");
        emit errorChanged();
        emit statusChanged();
        emit readyChanged();
        return;
    }
    m_error = takeString(OrangeFlex_LastError());
    m_progress = -1.0;
    m_ready = OrangeFlex_Ready() != 0;
    if (m_error.isEmpty())
        m_status.clear();
    else
        m_status = tr("Failed");
    refreshLoginFields();
    refreshAccountFields();
    applyLoggedIn(OrangeFlex_LoggedIn() != 0);
    emit errorChanged();
    emit progressChanged();
    emit statusChanged();
    emit readyChanged();
}

void ClientBridge::loginStartFinished(int rc)
{
    m_busy = false;
    emit busyChanged();
    refreshLoginFields();
    if (rc != 0) {
        m_error = takeString(OrangeFlex_LastError());
        if (m_error.isEmpty())
            m_error = tr("Could not send code");
        m_status = tr("Failed");
        emit errorChanged();
        emit statusChanged();
        return;
    }
    m_error.clear();
    m_progress = -1.0;
    m_status.clear();
    emit errorChanged();
    emit progressChanged();
    emit statusChanged();
    emit otpSent();
}

void ClientBridge::loginOtpFinished(int rc)
{
    m_busy = false;
    emit busyChanged();
    refreshLoginFields();
    if (rc < 0) {
        m_error = takeString(OrangeFlex_LastError());
        if (m_error.isEmpty())
            m_error = tr("Invalid code");
        m_status = tr("Failed");
        emit errorChanged();
        emit statusChanged();
        return;
    }
    m_error = takeString(OrangeFlex_LastError());
    m_progress = -1.0;
    if (m_error.isEmpty())
        m_status.clear();
    else
        m_status = tr("Failed");
    emit errorChanged();
    emit progressChanged();
    emit statusChanged();
    if (rc == 1) {
        emit needNextOtp();
        return;
    }
    if (rc == 2) {
        emit otpSent();
        return;
    }
    refreshAccountFields();
    applyLoggedIn(OrangeFlex_LoggedIn() != 0);
}

void ClientBridge::refreshFinished(int rc)
{
    m_busy = false;
    emit busyChanged();
    if (rc != 0) {
        m_error = takeString(OrangeFlex_LastError());
        if (m_error.isEmpty())
            m_error = tr("Refresh failed");
        m_status = tr("Failed");
        emit errorChanged();
        emit statusChanged();
        return;
    }
    m_error.clear();
    m_progress = -1.0;
    m_status.clear();
    refreshAccountFields();
    applyLoggedIn(OrangeFlex_LoggedIn() != 0);
    emit errorChanged();
    emit progressChanged();
    emit statusChanged();
}

void ClientBridge::logoutFinished(int rc)
{
    m_busy = false;
    emit busyChanged();
    if (rc != 0) {
        m_error = takeString(OrangeFlex_LastError());
        if (m_error.isEmpty())
            m_error = tr("Sign out failed");
        emit errorChanged();
        emit statusChanged();
        return;
    }
    m_error.clear();
    m_status.clear();
    refreshLoginFields();
    refreshAccountFields();
    applyLoggedIn(false);
    emit errorChanged();
    emit statusChanged();
}

void ClientBridge::wipeFinished(int rc)
{
    QSettings settings;
    settings.clear();
    settings.sync();
    m_networkAllowed = true;
    m_needsNetworkPrompt = false;
    m_ready = false;
    m_busy = false;
    applyLoggedIn(false);
    if (rc != 0) {
        m_error = takeString(OrangeFlex_LastError());
        if (m_error.isEmpty())
            m_error = tr("Delete failed");
        emit errorChanged();
    }
    QCoreApplication::quit();
}

void ClientBridge::loadListFinished(int rc)
{
    if (rc != 0) {
        finishAction(rc, tr("Could not load"));
        return;
    }
    finishAction(0, QString());
    applyListJson();
}

void ClientBridge::transferFinished(int rc)
{
    finishAction(rc, tr("Transfer failed"));
    if (rc == 0) {
        refreshAccountFields();
        emit transferSucceeded();
    }
}

void ClientBridge::topUpFinished(int rc)
{
    finishAction(rc, tr("BLIK top-up failed"));
    if (rc == 0) {
        refreshAccountFields();
        emit paymentSucceeded();
    }
}

void ClientBridge::withdrawFinished(int rc)
{
    finishAction(rc, tr("Withdraw failed"));
    if (rc == 0) {
        refreshAccountFields();
        emit withdrawSucceeded();
    }
}

void ClientBridge::markReadFinished(int rc)
{
    finishAction(rc, tr("Could not mark read"));
    if (rc == 0)
        loadList(QStringLiteral("messages"));
}
