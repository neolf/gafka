package kfs

import (
	"sync"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/Shopify/sarama"
	log "github.com/funkygao/log4go"
	"golang.org/x/net/context"
)

type File struct {
	sync.RWMutex
	attr fuse.Attr

	fs       *KafkaFS
	dir      *Dir
	consumer sarama.PartitionConsumer

	opened bool

	topic       string
	partitionId int32
}

func (f *File) Attr(ctx context.Context, o *fuse.Attr) error {
	f.RLock()
	defer f.RUnlock()

	log.Trace("File Attr, topic=%s, partitionId=%d", f.topic, f.partitionId)

	*o = f.attr

	// calculate size
	if !f.opened {
		if err := f.dir.reconnectKafkaIfNecessary(); err != nil {
			return err
		}

		latestOffset, err := f.dir.GetOffset(f.topic, f.partitionId, sarama.OffsetNewest)
		if err != nil {
			log.Error(err)

			return err
		}
		oldestOffset, err := f.dir.GetOffset(f.topic, f.partitionId, sarama.OffsetOldest)
		if err != nil {
			log.Error(err)

			return err
		}

		o.Size = uint64(latestOffset - oldestOffset)
	}

	return nil
}

func (f *File) Open(ctx context.Context, req *fuse.OpenRequest,
	resp *fuse.OpenResponse) (fs.Handle, error) {
	log.Trace("File Open, req=%#v, topic=%s, partitionId=%d", req,
		f.topic, f.partitionId)

	if err := f.reconsume(sarama.OffsetOldest); err != nil {
		return nil, err
	}

	// Allow kernel to use buffer cache
	resp.Flags &^= fuse.OpenDirectIO
	f.opened = true

	return f, nil
}

func (f *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	log.Trace("File Release, req=%#v, topic=%s, partitionId=%d", req,
		f.topic, f.partitionId)
	f.opened = false
	return f.consumer.Close()
}

func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	f.RLock()
	defer f.RUnlock()

	log.Trace("File ReadAll, topic=%s, partitionId=%d", f.topic, f.partitionId)

	out := make([]byte, 0)
	f.attr.Size = 0

LOOP:
	for {
		select {
		case msg := <-f.consumer.Messages():
			out = append(out, msg.Value...)
			out = append(out, '\n')
			f.attr.Size += uint64(len(msg.Value) + 1)

		case <-time.After(time.Second * 5):
			break LOOP
		}

	}

	return out, nil
}

func (f *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	f.RLock()
	defer f.RUnlock()

	log.Trace("File Read, req=%#v, topic=%s, partitionId=%d", req,
		f.topic, f.partitionId)

	// TODO req.Size, req.Offset
	offset := 0
	resp.Data = resp.Data[:req.Size]
	for {
		select {
		case msg := <-f.consumer.Messages():
			if offset+len(msg.Value) > req.Size {
				return nil
			}

			log.Trace("offset: %d, msg: %s, data: %s", offset, string(msg.Value), string(resp.Data))

			copy(resp.Data[offset:], msg.Value)
			offset += len(msg.Value)

		case <-time.After(time.Second * 5):
			return nil
		}

	}

	return nil
}

func (f *File) reconsume(offset int64) error {
	if err := f.dir.reconnectKafkaIfNecessary(); err != nil {
		return err
	}

	consumer, err := sarama.NewConsumerFromClient(f.dir.Client)
	if err != nil {
		log.Error(err)

		return err
	}

	cp, err := consumer.ConsumePartition(f.topic, f.partitionId, offset)
	if err != nil {
		log.Error(err)

		return err
	}

	f.consumer = cp
	return nil
}

func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	return fuse.EPERM
}
