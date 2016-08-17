count=$(pgrep -lf supervisord | grep '/data/www/bugle_provider/conf/test/supervisord.conf' | grep -v 'grep' | wc -l)

if [ $count -gt 0 ]; then
    /usr/bin/supervisorctl -c /data/www/bugle_provider/conf/test/supervisord.conf stop all
    pgrep -lf supervisord | grep '/data/www/bugle_provider/conf/test/supervisord.conf' | grep -v grep | awk '{print $1}' | xargs kill
    for i in `seq 5`; do
        sleep 2s
        pgrep -lf supervisord | grep '/data/www/bugle_provider/conf/test/supervisord.conf' | grep -v grep && continue
        exit 0
    done
exit 1
else
    echo "service is not running"
fi
