package org.ossrs.gmoryx.example.httpd;

import android.support.v7.app.AppCompatActivity;
import android.os.Bundle;
import android.widget.TextView;

import java.net.Inet6Address;
import java.net.InetAddress;
import java.net.NetworkInterface;
import java.net.SocketException;
import java.util.Enumeration;

import gmoryx.Gmoryx;
import gmoryx.HttpHandler;
import gmoryx.HttpRequest;
import gmoryx.HttpResponseWriter;

public class MainActivity extends AppCompatActivity {
    private TextView txtMain;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);

        txtMain = (TextView)findViewById(R.id.txtMain);
        txtMain.setText("Starting golang webserver...");
    }

    @Override
    protected void onResume() {
        super.onResume();

        Gmoryx.httpHandle("/", new HttpHandler() {
            @Override
            public void serveHTTP(HttpResponseWriter w, HttpRequest r) {
                Gmoryx.setHeader(w);

                try {
                    StringBuilder sb = new StringBuilder();
                    sb.append("<html>");
                    sb.append("Welcome to ");
                    sb.append("<a href='https://github.com/ossrs/go-oryx-lib/tree/master/gmoryx'>GMOryx(GoMobile Oryx)</a>");
                    sb.append("!");
                    w.write(sb.toString().getBytes());
                } catch (Exception e) {
                    e.printStackTrace();
                }
            }
        });

        try {
            Gmoryx.setServer("GMOryx/0.1");
            Gmoryx.httpListenAndServe(":8080", null);
            txtMain.setText("Web server: http://" + getHostIP() + ":8080\nPlease access from other machine.");
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    @Override
    protected void onPause() {
        super.onPause();

        try {
            Gmoryx.httpShutdown();
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    public static String getHostIP() {
        String hostIp = null;
        try {
            Enumeration nis = NetworkInterface.getNetworkInterfaces();
            InetAddress ia = null;
            while (nis.hasMoreElements()) {
                NetworkInterface ni = (NetworkInterface) nis.nextElement();
                Enumeration<InetAddress> ias = ni.getInetAddresses();
                while (ias.hasMoreElements()) {
                    ia = ias.nextElement();
                    if (ia instanceof Inet6Address) {
                        continue;// skip ipv6
                    }
                    String ip = ia.getHostAddress();
                    if (!"127.0.0.1".equals(ip)) {
                        hostIp = ia.getHostAddress();
                        break;
                    }
                }
            }
        } catch (SocketException e) {
            e.printStackTrace();
        }
        return hostIp;
    }
}
