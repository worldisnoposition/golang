# golang
temp file
public boolean write(String value, long lockTime) throws IOException {
        RandomAccessFile randomAccessFile = null;
        try {
            randomAccessFile = new RandomAccessFile(filename, "rw");
            FileChannel fileChannel = randomAccessFile.getChannel();
            int size = 500;
            MappedByteBuffer mappedByteBuffer = fileChannel.map(FileChannel.MapMode.READ_WRITE, 0, size);

            long time = System.currentTimeMillis();
            while (System.currentTimeMillis() - time < lockTime) {
                mappedByteBuffer.putInt(1);
                mappedByteBuffer.put(value.getBytes());
                mappedByteBuffer.putInt(READ_MODE);
                return true;
            }
            return false;
        } finally {
            if (randomAccessFile != null) {
                randomAccessFile.close();
            }
        }
    }

    public void read() throws IOException {
//        MappedByteBuffer mappedByteBuffer = null;
        StringBuilder sb = new StringBuilder();
        RandomAccessFile randomAccessFile = null;
        try {
            randomAccessFile = new RandomAccessFile(filename, "r");
            FileChannel fileChannel = randomAccessFile.getChannel();
            int size = 50;
            MappedByteBuffer mappedByteBuffer = fileChannel.map(FileChannel.MapMode.READ_ONLY, 0, size);
            while (true) {
                byte[] bb = new byte[size+1];
                while (mappedByteBuffer.hasRemaining()) {
                    Thread.sleep(1000L);
                    byte b = mappedByteBuffer.get();
                    bb[mappedByteBuffer.position()] = b;
//                    sb.append(new String(bb));
                    System.out.println(new String(bb));
                }
            }
//            return sb.toString();
        } catch (InterruptedException e) {
            e.printStackTrace();
        } finally {
            if (randomAccessFile != null) {
                randomAccessFile.close();
            }
        }
//        return null;
    }