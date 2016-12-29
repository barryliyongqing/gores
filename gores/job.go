package gores

import (
    "fmt"
    "time"
)

type Job struct {
    queue string
    payload map[string]interface{}
    resq *ResQ
    worker string
    enqueue_timestamp float64
}

func NewJob(queue string, payload map[string]interface{}, resq *ResQ, worker string) *Job {
    return &Job{
                queue: queue,
                payload: payload,
                resq: resq,
                worker: worker,
                enqueue_timestamp: payload["Enqueue_timestamp"].(float64),
                // Redis LPOP reply json, timestamp will be parsed to be float64
            }
}

func (job *Job) String() string {
    res := fmt.Sprintf("Job{%s} | %s ", job.queue, job.payload["Name"])
    return res
}

func (job *Job) Perform() error{
    struct_name := job.payload["Name"].(string)
    instance := StrToInstance(struct_name)
    args := job.payload["Args"].(map[string]interface{})

    metadata := make(map[string]interface{})
    for k, v := range args {
        metadata[k] = v
    }

    if job.enqueue_timestamp != 0 {
        metadata["enqueue_timestamp"] = job.enqueue_timestamp
    }
    metadata["failed"] = false
    //now, _ := strconv.Atoi(time.Now().Format("20060102150405"))
    now := time.Now().Unix()
    metadata["perfomed_timestamp"] = now

    err := InstancePerform(instance, args)
    if err != nil {
        metadata["failed"] = true
        if job.Retry(job.payload) {
            metadata["retried"] = true
        } else {
            metadata["retried"] = false
        }
        // InstanceAfterPerform() deal with metadata
    }
    return err
}

func (job *Job) Retry(payload map[string]interface{}) bool {
    _, toRetry := job.payload["Retry"]
    retry_every := job.payload["Retry_every"]
    if !toRetry || retry_every == nil {
        return false
    } else {
        now := job.resq.CurrentTime()
        retry_at := now + int64(retry_every.(float64))
        fmt.Printf("retry_at: %d\n", retry_at)
        err := job.resq.Enqueue_at(retry_at, payload)
        if err != nil {
            return false
        }
        return true
    }
}
