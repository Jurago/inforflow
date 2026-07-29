(function () {
    const { formatMbps, createIfaceCard, fetchSNMP } = window.Inforflow;

    function roleBadge(role) {
        return `<span class="badge badge-${role === 'ix' ? 'peer' : role === 'cache' ? 'cdn' : 'other'}">${role || '—'}</span>`;
    }

    async function update() {
        const snmp = await fetchSNMP();
        if (!snmp || !snmp.ok) {
            document.getElementById('snmp-sysname').textContent = 'SNMP offline';
            return;
        }

        document.getElementById('snmp-sysname').textContent = snmp.sys_name || '—';
        document.getElementById('snmp-uptime').textContent = snmp.uptime_human || '—';
        document.getElementById('snmp-cpu').textContent = (snmp.cpu_pct || 0).toFixed(1) + '%';
        document.getElementById('snmp-mem').textContent = (snmp.mem_pct || 0).toFixed(1) + '%';
        document.getElementById('snmp-uplink').textContent =
            formatMbps(snmp.uplink_in_mbps) + ' / ' + formatMbps(snmp.uplink_out_mbps);
        document.getElementById('snmp-uplink-util').textContent =
            'utilização ' + (snmp.uplink_util_pct || 0).toFixed(2) + '%';

        const bars = document.getElementById('iface-bars');
        if (bars) {
            const list = (snmp.interfaces || [])
                .filter(i => i.oper_status === 1 && (i.in_mbps + i.out_mbps) > 0.5)
                .slice(0, 12);
            const max = Math.max(...list.map(i => Math.max(i.in_mbps, i.out_mbps)), 1);
            bars.innerHTML = list.map(i => {
                const label = (i.alias || i.name).substring(0, 22);
                return `<div class="iface-bar-row">
                    <div class="iface-bar-label" title="${i.name}">${label}</div>
                    <div class="iface-bar-tracks">
                        <div class="iface-bar in" style="width:${(i.in_mbps / max * 100)}%"></div>
                        <div class="iface-bar out" style="width:${(i.out_mbps / max * 100)}%"></div>
                    </div>
                    <div class="iface-bar-vals">${formatMbps(i.in_mbps)} ↓ · ${formatMbps(i.out_mbps)} ↑</div>
                </div>`;
            }).join('');
        }

        const topIn = document.getElementById('snmp-top-in');
        const topOut = document.getElementById('snmp-top-out');
        if (topIn) topIn.innerHTML = (snmp.top_in || []).slice(0, 6).map(createIfaceCard).join('');
        if (topOut) topOut.innerHTML = (snmp.top_out || []).slice(0, 6).map(createIfaceCard).join('');

        const tbody = document.getElementById('snmp-iface-body');
        if (tbody) {
            const rows = (snmp.interfaces || [])
                .filter(i => i.oper_status === 1 || (i.alias && i.alias.length))
                .slice(0, 40);
            tbody.innerHTML = rows.map(i => {
                const util = Math.max(i.in_util_pct || 0, i.out_util_pct || 0);
                const st = i.oper_status === 1 ? '<span class="badge badge-inbound">up</span>' : '<span class="badge badge-other">down</span>';
                return `<tr>
                    <td>${i.index}</td>
                    <td>${i.name}</td>
                    <td>${i.alias || '—'}</td>
                    <td>${roleBadge(i.role)}</td>
                    <td>${st}</td>
                    <td>${i.speed_mbps || 0}</td>
                    <td>${formatMbps(i.in_mbps)}</td>
                    <td>${formatMbps(i.out_mbps)}</td>
                    <td>${util.toFixed(2)}%</td>
                </tr>`;
            }).join('');
        }
    }

    setInterval(update, 5000);
    update();
})();
