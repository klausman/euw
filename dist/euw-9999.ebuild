# Copyright 1999-2018 Gentoo Foundation
# Distributed under the terms of the GNU General Public License v2

EAPI=6
EGO_PN=github.com/klausman/euw

if [[ ${PV} = *9999* ]]; then
    inherit golang-vcs
else
    KEYWORDS="~amd64 ~x86"
    EGIT_COMMIT=v${PV}
    SRC_URI="https://${EGO_PN}/archive/${EGIT_COMMIT}.tar.gz -> ${P}.tar.gz"
    inherit golang-vcs-snapshot
fi
inherit golang-build

DESCRIPTION="A simple wrapper around edacutil to make it like mcelog"
HOMEPAGE="https://github.com/klausman/euw"

LICENSE="Apache-2.0"
SLOT="0"

DEPEND="sys-apps/edac-utils"
RDEPEND="${DEPEND}"

src_install() {
	dobin ${PN}

	doinitd init/${PN}.openrc
	doconfd init/${PN}.default

	declare -a DOCS
	DOCS+=(src/${EGO_PN}/README.md)
	einstalldocs
}
