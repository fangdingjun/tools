#!/usr/bin/python

import zipfile
import os


def unzip(filename, output_dir=".", filename_encode="gbk"):
    fp = zipfile.ZipFile(filename, "r")

    # get name list
    nlist = fp.namelist()

    for name in nlist:

        # decode filename
        n1 = name.decode(filename_encode)

        # ignore directory
        if n1.endswith("/"):
            continue

        # full filename
        n2 = os.path.join(output_dir, n1)

        # directory name
        d = os.path.dirname(n2)
        if not os.path.isdir(d):
            os.makedirs(d, 0o755)

        # extra content
        with open(n2, "wb") as fp1:
            fp1.write(fp.read(name))

    fp.close()


if __name__ == "__main__":
    #unzip("alipaydirect.zip", "a", "gbk")
    import sys
    if len(sys.argv) == 1:
        print """Usage:\n\t%s zipfile [output directory] [filename encoding]""" % (
            os.path.basename(sys.argv[0]),)
        sys.exit(1)
    fn = sys.argv[1].decode("utf-8")
    import os
    out, _ = os.path.splitext(os.path.basename(fn))
    code = "gbk"
    if len(sys.argv) > 2:
        out = sys.argv[2]
    if len(sys.argv) > 3:
        code = sys.argv[3]
    unzip(fn, out, code)
