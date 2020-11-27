/*
 * Copyright © 2020 Camilo Hernández <me@camiloh.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 */

package beekeeper

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"log"
	"math"
	"runtime"
	"time"
)

// jobResultCallback is the callback for the JobResult operation.
func jobResultCallback(s *Server, conn Conn, msg Message) {
	s.checkAwaited(msg)
}

// transferStatusCallback is the callback for the JobTransferAcknowledge and JobTransferFailed operations.
func transferStatusCallback(s *Server, conn Conn, msg Message) {
	s.checkAwaited(msg)
}

// statusCallback is the callback for the Status operation.
func statusCallback(s *Server, conn Conn, msg Message) {
	ni := NodeInfo{}

	// CPU Usage
	usageSlice, err := cpu.Percent(time.Second, false)
	if err == nil && len(usageSlice) > 0 {
		ni.Usage = float32(usageSlice[0])
	}

	// CPU Temp
	ni.CPUTemp = getCPUTemp()

	err = conn.send(Message{NodeInfo: ni})
	if err != nil {
		log.Println("Error while responding to Status Request:", err.Error())
		return
	}
}

// jobTransferCallback is the callback for the JobTransfer operation.
func jobTransferCallback(s *Server, conn Conn, msg Message) {
	log.Println("Transferring new job from node", msg.From)

	folderPath := ".beekeeper"
	err := createFolderIfNotExist(folderPath)
	if err != nil {
		log.Println("Unable to create beekeeper folder:", err.Error())
		respondTransferError(conn, err.Error())

		return
	}

	if len(msg.Data) == 0 {
		log.Println("Unable to save job data: empty data field")
		respondTransferError(conn, "empty data field")

		return
	}

	binPath := folderPath + "/job.bin"
	err = saveBinary(binPath, msg.Data)
	if err != nil {
		log.Println("Unable to save job data:", err.Error())
		respondTransferError(conn, err.Error())

		return
	}

	err = conn.send(Message{Operation: OperationTransferAcknowledge})
	if err != nil {
		log.Println("Error while acknowledging transfer:", err.Error())

		return
	}

	log.Println("New job transferred successfully")
}

// jobExecuteCallback is the callback for the JobExecute operation.
func jobExecuteCallback(s *Server, conn Conn, msg Message) {
	task, err := decodeTask(msg.Data)
	if err != nil {
		log.Println("Unable to read task data:", err.Error())
		return
	}

	log.Println("Executing task", task.UUID, "from node", msg.From)

	s.Status = StatusWorking

	res, err := runLocalJob(task)
	if err != nil {
		errMsg := "Unable to run job: " + err.Error()
		log.Println(errMsg)

		res = Result{UUID: task.UUID, Error: errMsg}
	}

	log.Println("Ran task", task.UUID, "successfully")

	s.Status = StatusIDLE

	resBytes, err := res.encode()
	if err != nil {
		log.Println("Unable to encode response:", err.Error())
		return
	}

	err = conn.send(Message{
		Operation: OperationJobResult,
		Data:      resBytes,
	})
	if err != nil {
		log.Println("Error while sending Result to job Request:", err.Error())
		return
	}
}

// respondTransferError is a shorthand for sending a TransferFailed operation to the remote node.
func respondTransferError(conn Conn, errMsg string) {
	err := conn.send(Message{Operation: OperationTransferFailed, Data: []byte(errMsg)})
	if err != nil {
		log.Println("Error while reporting transfer error:", err.Error())
	}
}

// getCPUTemp tries it's best to find the CPU temperature for the host OS
func getCPUTemp() float32 {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		return 0
	}

	switch runtime.GOOS {
	case "linux":
		sensorKeyTemplate := "coretemp_core%d_input"
		coreNum := 0

		var coreTempsTotal float64
		for {
			key := fmt.Sprintf(sensorKeyTemplate, coreNum)
			for _, sensor := range temps {
				if sensor.SensorKey == key {
					coreTempsTotal += sensor.Temperature

					coreNum += 1
					continue
				}
			}

			break
		}

		if coreNum == 0 { // No sensor found
			return 0
		}

		average := coreTempsTotal / float64(coreNum)
		return float32(math.Round(average*10) / 10) // Once decimal place

	case "darwin":
		key := "TC0P"
		for _, sensor := range temps {
			if sensor.SensorKey == key {
				return float32(math.Round(sensor.Temperature*10) / 10)
			}
		}

		return 0 // Not found

	default:
		if len(temps) > 0 {
			// Return the highest value (probably the CPU)
			var n, biggest float32

			for _, v := range temps {
				temp := float32(v.Temperature)
				if temp > n {
					n = temp
					biggest = n
				}
			}

			return biggest
		}

		return 0
	}
}
