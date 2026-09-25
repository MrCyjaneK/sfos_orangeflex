Name: harbour-orangeflex

Summary: Orange Flex
Version: 0.1
Release: 1
Group: Qt/Qt
License: BSD
Source0: %{name}-%{version}.tar.bz2
Requires: sailfishsilica-qt5 >= 0.10.9
BuildRequires: pkgconfig(sailfishapp) >= 1.0.2
BuildRequires: pkgconfig(Qt5Core)
BuildRequires: pkgconfig(Qt5Gui)
BuildRequires: pkgconfig(Qt5Qml)
BuildRequires: pkgconfig(Qt5Quick)
BuildRequires: desktop-file-utils
BuildRequires: sailfishsilica-qt5 >= 0.10.9

%description
Unofficial Sailfish OS client for Orange Flex.

%prep
%autosetup -n %{name}-%{version}

%build
mkdir -p lib
%qmake5
%make_build

%install
%qmake5_install
desktop-file-install --delete-original \
    --dir %{buildroot}%{_datadir}/applications \
    %{buildroot}%{_datadir}/applications/*.desktop

%files
%{_bindir}/%{name}
%{_datadir}/%{name}
%{_datadir}/applications/%{name}.desktop
%{_datadir}/icons/hicolor/*/apps/%{name}.png
