#ifndef ORANGEFLEX_H
#define ORANGEFLEX_H

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef void (*orangeflex_progress_fn)(const char *step, const char *message, int64_t bytes, int64_t total);

void OrangeFlex_SetProgressHandler(orangeflex_progress_fn fn);
void OrangeFlex_SetCacheDir(const char *dir);
void OrangeFlex_SetNetworkAllowed(int allow);
int OrangeFlex_NetworkAllowed(void);
int OrangeFlex_NetworkPromptNeeded(void);
int OrangeFlex_Init(void);
int OrangeFlex_Ready(void);
int OrangeFlex_LoggedIn(void);
char *OrangeFlex_LastError(void);
void OrangeFlex_Free(char *p);

int OrangeFlex_LoginStart(const char *to);
int OrangeFlex_LoginSubmitOTP(const char *otp);
int OrangeFlex_LoginTimeout(void);
char *OrangeFlex_PendingEmailMask(void);
char *OrangeFlex_PendingMSISDN(void);
int OrangeFlex_Refresh(void);
int OrangeFlex_Logout(void);
int OrangeFlex_WipeData(void);
void OrangeFlex_SetLanguage(const char *lang);
char *OrangeFlex_Language(void);

char *OrangeFlex_DisplayName(void);
char *OrangeFlex_FirstName(void);
char *OrangeFlex_LastName(void);
char *OrangeFlex_Email(void);
char *OrangeFlex_MSISDN(void);
char *OrangeFlex_OfferingName(void);
char *OrangeFlex_ProductStatus(void);
char *OrangeFlex_RenewalDate(void);
char *OrangeFlex_RenewalTime(void);
char *OrangeFlex_LeftGB(void);
char *OrangeFlex_GrantGB(void);
char *OrangeFlex_WalletAmount(void);
char *OrangeFlex_DataSafeGB(void);

int OrangeFlex_SIMCount(void);
char *OrangeFlex_SIMHierarchy(int index);
char *OrangeFlex_SIMType(int index);
char *OrangeFlex_SIMLabel(int index);
int OrangeFlex_MultisimUsed(void);
int OrangeFlex_MultisimTotal(void);
int OrangeFlex_MultisimAvailable(void);

char *OrangeFlex_GroupType(void);
int OrangeFlex_MemberCount(void);
char *OrangeFlex_MemberAlias(int index);
char *OrangeFlex_MemberRole(int index);
char *OrangeFlex_MemberMSISDN(int index);
char *OrangeFlex_MemberStatus(int index);
char *OrangeFlex_MemberLeftGB(int index);
char *OrangeFlex_MemberGrantGB(int index);

char *OrangeFlex_ExtraJSON(void);
int OrangeFlex_LoadList(const char *name);
char *OrangeFlex_ListJSON(void);
char *OrangeFlex_ListName(void);
int OrangeFlex_TransferData(const char *to, const char *value, const char *message);
int OrangeFlex_TopUpBlik(const char *amount, const char *authCode);
int OrangeFlex_WithdrawDataSafe(const char *amount);
int OrangeFlex_MarkMessagesRead(void);

#ifdef __cplusplus
}
#endif

#endif
