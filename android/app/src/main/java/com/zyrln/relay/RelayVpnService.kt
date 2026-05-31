package com.zyrln.relay

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.net.ProxyInfo
import android.net.VpnService
import android.os.Build
import android.os.ParcelFileDescriptor
import android.util.Log
import androidx.core.app.NotificationCompat
import mobile.Mobile

class RelayVpnService : VpnService() {

    private var vpnInterface: ParcelFileDescriptor? = null
    @Volatile private var startGeneration = 0

    companion object {
        const val TAG = "RelayVpnService"
        const val ACTION_START = "com.zyrln.relay.START"
        const val ACTION_STOP = "com.zyrln.relay.STOP"
        const val ACTION_STARTED = "com.zyrln.relay.STARTED"
        const val ACTION_ERROR = "com.zyrln.relay.ERROR"
        const val ACTION_STOPPED = "com.zyrln.relay.STOPPED"
        const val EXTRA_URL = "url"
        const val EXTRA_KEY = "key"
        const val EXTRA_ERROR = "error"
        const val NOTIF_ID = 1
        const val CHANNEL_ID = "zyrln_vpn"
        private const val PROXY_PORT = 8085
        private const val VPN_ADDRESS = "10.99.0.2"
        private const val VPN_PREFIX = 24
        private const val VPN_MTU = 1500
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (intent?.action == ACTION_STOP) {
            stopRelay()
            return START_NOT_STICKY
        }

        val url = intent?.getStringExtra(EXTRA_URL) ?: return START_NOT_STICKY
        val key = intent.getStringExtra(EXTRA_KEY) ?: return START_NOT_STICKY

        startGeneration++
        if (Mobile.isRunning()) {
            Mobile.setSocketProtector(null)
            Mobile.stop()
            vpnInterface?.close()
            vpnInterface = null
        }

        startForeground(NOTIF_ID, buildNotification(url.isEmpty()))
        val gen = startGeneration
        val vpnService = this
        Mobile.setSocketProtector(object : mobile.SocketProtector {
            override fun protect(p0: Long): Boolean = vpnService.protect(p0.toInt())
        })
        Thread {
            startRelay(url, key, gen)
        }.start()
        return START_STICKY
    }

    private fun startRelay(url: String, key: String, generation: Int) {
        if (generation != startGeneration) return

        val err = if (url.isEmpty()) {
            Mobile.startDirect("127.0.0.1:$PROXY_PORT")
        } else {
            Mobile.startTunnel(url, key, "127.0.0.1:$PROXY_PORT")
        }

        if (generation != startGeneration) return
        if (err.isNotEmpty()) {
            Log.e(TAG, "relay start failed: $err")
            failStart(getString(R.string.error_relay_start_failed, err), generation)
            return
        }
        Log.i(TAG, "relay started on 127.0.0.1:$PROXY_PORT (directOnly=${url.isEmpty()})")
        if (generation != startGeneration) {
            Mobile.setSocketProtector(null)
            Mobile.stop()
            return
        }

        try {
            // TUN routes all IPv4; Go forwarder dials tunnel/direct per flow. HTTP proxy kept for proxy-aware apps.
            vpnInterface = Builder()
                .setSession(getString(R.string.vpn_session_name))
                .setMtu(VPN_MTU)
                .addAddress(VPN_ADDRESS, VPN_PREFIX)
                .addRoute("0.0.0.0", 0)
                .addDnsServer("8.8.8.8")
                .addDnsServer("1.1.1.1")
                .setHttpProxy(ProxyInfo.buildDirectProxy("127.0.0.1", PROXY_PORT))
                .establish()
            if (vpnInterface == null) {
                failStart(getString(R.string.error_vpn_permission), generation)
                return
            }
            if (generation != startGeneration) {
                Mobile.setSocketProtector(null)
                Mobile.stop()
                vpnInterface?.close()
                vpnInterface = null
                return
            }
            val tunErr = Mobile.attachTUN(vpnInterface!!.fd.toLong())
            if (tunErr.isNotEmpty()) {
                Log.e(TAG, "TUN forwarder failed: $tunErr")
                failStart(getString(R.string.error_relay_start_failed, tunErr), generation)
                return
            }
            Log.i(
                TAG,
                "TUN forwarder active fd=${vpnInterface!!.fd} routes=0.0.0.0/0 proxy=127.0.0.1:$PROXY_PORT"
            )
            sendBroadcast(Intent(ACTION_STARTED))
        } catch (e: Exception) {
            Log.e(TAG, "VPN establish failed: ${e.message}")
            Mobile.setSocketProtector(null)
            Mobile.stop()
            vpnInterface?.close()
            vpnInterface = null
            stopForeground(STOP_FOREGROUND_REMOVE)
            stopSelf()
        }
    }

    private fun failStart(message: String, generation: Int) {
        if (generation != startGeneration) return
        Log.e(TAG, message)
        Mobile.setSocketProtector(null)
        Mobile.stop()
        vpnInterface?.close()
        vpnInterface = null
        stopForeground(STOP_FOREGROUND_REMOVE)
        sendBroadcast(Intent(ACTION_ERROR).putExtra(EXTRA_ERROR, message))
        stopSelf()
    }

    private fun stopRelay() {
        startGeneration++
        Log.i(TAG, "stopping relay")
        Mobile.setSocketProtector(null)
        Mobile.stop()
        vpnInterface?.close()
        vpnInterface = null
        stopForeground(STOP_FOREGROUND_REMOVE)
        sendBroadcast(Intent(ACTION_STOPPED))
        stopSelf()
    }

    override fun onDestroy() {
        Mobile.setSocketProtector(null)
        Mobile.stop()
        vpnInterface?.close()
        vpnInterface = null
        sendBroadcast(Intent(ACTION_STOPPED))
        super.onDestroy()
    }

    private fun buildNotification(directOnly: Boolean): Notification {
        createNotificationChannel()
        val openIntent = PendingIntent.getActivity(
            this, 0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        val stopIntent = PendingIntent.getService(
            this, 0,
            Intent(this, RelayVpnService::class.java).apply { action = ACTION_STOP },
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        )
        val title = if (directOnly) {
            getString(R.string.direct_notification_title)
        } else {
            getString(R.string.vpn_notification_title)
        }
        val text = if (directOnly) {
            getString(R.string.direct_notification_text)
        } else {
            getString(R.string.vpn_notification_text)
        }
        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle(title)
            .setContentText(text)
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentIntent(openIntent)
            .addAction(android.R.drawable.ic_media_pause, getString(R.string.btn_disconnect), stopIntent)
            .setOngoing(true)
            .build()
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                getString(R.string.channel_name_vpn),
                NotificationManager.IMPORTANCE_LOW
            )
            getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
        }
    }
}
