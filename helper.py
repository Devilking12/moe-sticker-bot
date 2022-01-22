from threading import Timer
import glob
import os
import time
import shutil
import main

def start_timer_userdata_gc():
    timer = Timer(43200, start_timer_userdata_gc)
    timer.start()

    data_dir = main.DATA_DIR
    if not os.path.exists(data_dir):
        os.mkdir(data_dir)
        return
    for d in glob.glob(os.path.join(data_dir, "*")):
        mtime = os.path.getmtime(d)
        nowtime = time.time()
        if nowtime - mtime > 43200:
            shutil.rmtree(d, ignore_errors=True)