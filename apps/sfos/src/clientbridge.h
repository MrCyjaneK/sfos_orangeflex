#ifndef CLIENTBRIDGE_H
#define CLIENTBRIDGE_H

#include <QObject>
#include <QString>
#include <QVariantList>
#include <QVariantMap>

class ClientBridge : public QObject
{
    Q_OBJECT
    Q_PROPERTY(bool networkAllowed READ networkAllowed NOTIFY networkAllowedChanged)
    Q_PROPERTY(bool needsNetworkPrompt READ needsNetworkPrompt NOTIFY needsNetworkPromptChanged)
    Q_PROPERTY(bool busy READ busy NOTIFY busyChanged)
    Q_PROPERTY(bool ready READ ready NOTIFY readyChanged)
    Q_PROPERTY(bool loggedIn READ loggedIn NOTIFY loggedInChanged)
    Q_PROPERTY(QString status READ status NOTIFY statusChanged)
    Q_PROPERTY(QString error READ error NOTIFY errorChanged)
    Q_PROPERTY(double progress READ progress NOTIFY progressChanged)
    Q_PROPERTY(int loginTimeout READ loginTimeout NOTIFY loginChanged)
    Q_PROPERTY(QString pendingEmailMask READ pendingEmailMask NOTIFY loginChanged)
    Q_PROPERTY(QString pendingMsisdn READ pendingMsisdn NOTIFY loginChanged)
    Q_PROPERTY(QString language READ language NOTIFY languageChanged)
    Q_PROPERTY(QString displayName READ displayName NOTIFY dataChanged)
    Q_PROPERTY(QString firstName READ firstName NOTIFY dataChanged)
    Q_PROPERTY(QString lastName READ lastName NOTIFY dataChanged)
    Q_PROPERTY(QString email READ email NOTIFY dataChanged)
    Q_PROPERTY(QString msisdn READ msisdn NOTIFY dataChanged)
    Q_PROPERTY(QString offeringName READ offeringName NOTIFY dataChanged)
    Q_PROPERTY(QString productStatus READ productStatus NOTIFY dataChanged)
    Q_PROPERTY(QString renewalDate READ renewalDate NOTIFY dataChanged)
    Q_PROPERTY(QString renewalTime READ renewalTime NOTIFY dataChanged)
    Q_PROPERTY(QString leftGB READ leftGB NOTIFY dataChanged)
    Q_PROPERTY(QString grantGB READ grantGB NOTIFY dataChanged)
    Q_PROPERTY(QString walletAmount READ walletAmount NOTIFY dataChanged)
    Q_PROPERTY(QString dataSafeGB READ dataSafeGB NOTIFY dataChanged)
    Q_PROPERTY(QString groupType READ groupType NOTIFY dataChanged)
    Q_PROPERTY(int multisimUsed READ multisimUsed NOTIFY dataChanged)
    Q_PROPERTY(int multisimTotal READ multisimTotal NOTIFY dataChanged)
    Q_PROPERTY(int multisimAvailable READ multisimAvailable NOTIFY dataChanged)
    Q_PROPERTY(QVariantList sims READ sims NOTIFY dataChanged)
    Q_PROPERTY(QVariantList members READ members NOTIFY dataChanged)
    Q_PROPERTY(QString roamingLeftGB READ roamingLeftGB NOTIFY dataChanged)
    Q_PROPERTY(QString roamingGrantGB READ roamingGrantGB NOTIFY dataChanged)
    Q_PROPERTY(QVariantList dataSafePeriods READ dataSafePeriods NOTIFY dataChanged)
    Q_PROPERTY(QVariantList meters READ meters NOTIFY dataChanged)
    Q_PROPERTY(QVariantList paymentMethods READ paymentMethods NOTIFY dataChanged)
    Q_PROPERTY(QVariantList flags READ flags NOTIFY dataChanged)
    Q_PROPERTY(QVariantList currentList READ currentList NOTIFY listChanged)
    Q_PROPERTY(QString listName READ listName NOTIFY listChanged)

public:
    explicit ClientBridge(QObject *parent = 0);

    bool networkAllowed() const { return m_networkAllowed; }
    bool needsNetworkPrompt() const { return m_needsNetworkPrompt; }
    bool busy() const { return m_busy; }
    bool ready() const { return m_ready; }
    bool loggedIn() const { return m_loggedIn; }
    QString status() const { return m_status; }
    QString error() const { return m_error; }
    double progress() const { return m_progress; }
    int loginTimeout() const { return m_loginTimeout; }
    QString pendingEmailMask() const { return m_pendingEmailMask; }
    QString pendingMsisdn() const { return m_pendingMsisdn; }
    QString language() const { return m_language; }
    QString displayName() const { return m_displayName; }
    QString firstName() const { return m_firstName; }
    QString lastName() const { return m_lastName; }
    QString email() const { return m_email; }
    QString msisdn() const { return m_msisdn; }
    QString offeringName() const { return m_offeringName; }
    QString productStatus() const { return m_productStatus; }
    QString renewalDate() const { return m_renewalDate; }
    QString renewalTime() const { return m_renewalTime; }
    QString leftGB() const { return m_leftGB; }
    QString grantGB() const { return m_grantGB; }
    QString walletAmount() const { return m_walletAmount; }
    QString dataSafeGB() const { return m_dataSafeGB; }
    QString groupType() const { return m_groupType; }
    int multisimUsed() const { return m_multisimUsed; }
    int multisimTotal() const { return m_multisimTotal; }
    int multisimAvailable() const { return m_multisimAvailable; }
    QVariantList sims() const { return m_sims; }
    QVariantList members() const { return m_members; }
    QString roamingLeftGB() const { return m_roamingLeftGB; }
    QString roamingGrantGB() const { return m_roamingGrantGB; }
    QVariantList dataSafePeriods() const { return m_dataSafePeriods; }
    QVariantList meters() const { return m_meters; }
    QVariantList paymentMethods() const { return m_paymentMethods; }
    QVariantList flags() const { return m_flags; }
    QVariantList currentList() const { return m_currentList; }
    QString listName() const { return m_listName; }

public slots:
    void setNetworkAllowed(bool allow);
    void startInit();
    void loginStart(const QString &to);
    void loginSubmitOtp(const QString &otp);
    void refreshAccount();
    void logout();
    void setLanguage(const QString &lang);
    void wipeAndExit();
    void loadList(const QString &name);
    void transferData(const QString &to, const QString &value, const QString &message);
    void topUpBlik(const QString &amount, const QString &authCode);
    void withdrawDataSafe(const QString &amount);
    void markMessagesRead();
    Q_INVOKABLE void onProgress(const QString &step, const QString &message, qint64 bytes, qint64 total);
    Q_INVOKABLE void initFinished(int rc);
    Q_INVOKABLE void loginStartFinished(int rc);
    Q_INVOKABLE void loginOtpFinished(int rc);
    Q_INVOKABLE void refreshFinished(int rc);
    Q_INVOKABLE void logoutFinished(int rc);
    Q_INVOKABLE void wipeFinished(int rc);
    Q_INVOKABLE void loadListFinished(int rc);
    Q_INVOKABLE void transferFinished(int rc);
    Q_INVOKABLE void topUpFinished(int rc);
    Q_INVOKABLE void withdrawFinished(int rc);
    Q_INVOKABLE void markReadFinished(int rc);

signals:
    void networkAllowedChanged();
    void needsNetworkPromptChanged();
    void busyChanged();
    void readyChanged();
    void loggedInChanged();
    void statusChanged();
    void errorChanged();
    void progressChanged();
    void loginChanged();
    void languageChanged();
    void dataChanged();
    void otpSent();
    void needNextOtp();
    void listChanged();
    void transferSucceeded();
    void paymentSucceeded();
    void withdrawSucceeded();

private:
    QString takeString(char *p) const;
    void beginBusy(const QString &status);
    void refreshLoginFields();
    void refreshAccountFields();
    void applyLoggedIn(bool value);
    void applyExtraJson();
    void applyListJson();
    void finishAction(int rc, const QString &fallback);

    bool m_networkAllowed;
    bool m_needsNetworkPrompt;
    bool m_busy;
    bool m_ready;
    bool m_loggedIn;
    QString m_status;
    QString m_error;
    double m_progress;
    int m_loginTimeout;
    QString m_pendingEmailMask;
    QString m_pendingMsisdn;
    QString m_language;
    QString m_displayName;
    QString m_firstName;
    QString m_lastName;
    QString m_email;
    QString m_msisdn;
    QString m_offeringName;
    QString m_productStatus;
    QString m_renewalDate;
    QString m_renewalTime;
    QString m_leftGB;
    QString m_grantGB;
    QString m_walletAmount;
    QString m_dataSafeGB;
    QString m_groupType;
    int m_multisimUsed;
    int m_multisimTotal;
    int m_multisimAvailable;
    QVariantList m_sims;
    QVariantList m_members;
    QString m_roamingLeftGB;
    QString m_roamingGrantGB;
    QVariantList m_dataSafePeriods;
    QVariantList m_meters;
    QVariantList m_paymentMethods;
    QVariantList m_flags;
    QVariantList m_currentList;
    QString m_listName;
};

#endif
