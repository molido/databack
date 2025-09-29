/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	operatormolidocomv1beta1 "github.com/molido/databack-operator/api/v1beta1"
)

// DatabackReconciler reconciles a Databack object
type DatabackReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	BackupQueue map[string]operatormolidocomv1beta1.Databack
	Wg          sync.WaitGroup
	Tickers     []*time.Ticker
	lock        sync.RWMutex
}

func (r *DatabackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	fmt.Println("Hello world Reconcile")

	//find resource

	var databackK8s operatormolidocomv1beta1.Databack
	err := r.Client.Get(ctx, req.NamespacedName, &databackK8s)
	if err != nil {
		fmt.Println("Get from k8s")

		if errors.IsNotFound(err) {
			//todo delete
			r.DeleteQueue(databackK8s)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if lastDataback, ok := r.BackupQueue[databackK8s.Name]; ok {
		isEqual := reflect.DeepEqual(lastDataback.Spec, databackK8s.Spec)
		if isEqual {
			return ctrl.Result{}, nil
		}
	}
	//todo create/update
	r.AddQueue(databackK8s)
	return ctrl.Result{}, nil
}

func (r *DatabackReconciler) StopLoop() {
	for _, ticker := range r.Tickers {
		if ticker != nil {
			ticker.Stop()
		}
	}

}
func (r *DatabackReconciler) StartLoop() {
	for _, databack := range r.BackupQueue {
		if !databack.Spec.Enable {
			databack.Status.Active = false
			r.UpdateStatus(databack)
			continue
		}
		delay := r.getDelaySeconds(databack.Spec.StartTime)
		fmt.Println("Start Loop")
		databack.Status.Active = true
		nextTime := r.getNextTime(delay.Seconds())
		databack.Status.NextTime = nextTime.Unix()
		r.UpdateStatus(databack)
		ticker := time.NewTicker(time.Duration(delay))
		r.Tickers = append(r.Tickers, ticker)
		r.Wg.Add(1)
		go func(databack operatormolidocomv1beta1.Databack) {
			for {
				<-ticker.C
				//backup
				//reset ticker
				ticker.Reset(time.Duration(databack.Spec.Period) * time.Minute)
				databack.Status.Active = true
				databack.Status.NextTime = r.getNextTime(float64(databack.Spec.Period) * 60).Unix()
				err := r.DumpMySql(databack)
				if err != nil {
					databack.Status.LastBackupResult = fmt.Sprintf("databack failed %v", err)
					fmt.Printf("databack failed %v\n", err)
				} else {
					databack.Status.LastBackupResult = "databack successful"
					fmt.Println("databack successful")
				}
				r.UpdateStatus(databack)
			}
		}(databack)

	}
	r.Wg.Wait()

}

func (r *DatabackReconciler) getDelaySeconds(startTime string) time.Duration {
	times := strings.Split(startTime, ":")
	expectedHour, _ := strconv.Atoi(times[0])
	expectedMin, _ := strconv.Atoi(times[1])
	now := time.Now().Truncate(time.Second)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.Add(24 * time.Hour)
	var seconds int
	expectedDuration := time.Hour*time.Duration(expectedHour) + time.Minute*time.Duration(expectedMin)
	curDuration := time.Hour*time.Duration(now.Hour()) + time.Minute*time.Duration(now.Minute())
	if curDuration >= expectedDuration {
		//tomorrow
		seconds = int(todayEnd.Add(expectedDuration).Sub(now).Seconds())
	} else {
		//today
		seconds = int(todayStart.Add(expectedDuration).Sub(now).Seconds())
	}
	return time.Duration(seconds)
}

func (r *DatabackReconciler) DeleteQueue(databack operatormolidocomv1beta1.Databack) {
	delete(r.BackupQueue, databack.Name)
	r.StartLoop()
	go r.StartLoop()
}

func (r *DatabackReconciler) AddQueue(databack operatormolidocomv1beta1.Databack) {
	if r.BackupQueue == nil {
		r.BackupQueue = make(map[string]operatormolidocomv1beta1.Databack)
	}
	r.BackupQueue[databack.Name] = databack
	r.StartLoop()
	go r.StartLoop()

}

func (r *DatabackReconciler) UpdateStatus(backup operatormolidocomv1beta1.Databack) {
	r.lock.Lock()
	defer r.lock.Unlock()
	ctx := context.TODO()
	namespacedName := types.NamespacedName{
		Name:      backup.Name,
		Namespace: backup.Namespace,
	}
	var dataBackupK8s operatormolidocomv1beta1.Databack
	err := r.Get(ctx, namespacedName, &dataBackupK8s)
	if err != nil {
		return
	}

	dataBackupK8s.Status = backup.Status
	err = r.Client.Status().Update(ctx, &dataBackupK8s)
	if err != nil {
		return
	}

}

// getNextTime 根据周期(秒)计算下次执行时间
func (r *DatabackReconciler) getNextTime(periodSeconds float64) time.Time {
	now := time.Now()
	return now.Add(time.Duration(periodSeconds) * time.Second)
}

func (r *DatabackReconciler) DumpMySql(backup operatormolidocomv1beta1.Databack) error {
	host := backup.Spec.Origin.Host
	port := backup.Spec.Origin.Port
	name := backup.Spec.Origin.Username
	pw := backup.Spec.Origin.Password
	now := time.Now()
	fmt.Println("DumpMySql Start")
	backupDate := fmt.Sprintf("%02d-%02d", now.Month(), now.Day())
	folderPath := fmt.Sprintf("/tmp/%s/%s/", backup.Name, backupDate)
	if _, err := os.Stat(folderPath); err != nil {
		if errx := os.MkdirAll(folderPath, 0700); errx == nil {
			fmt.Println("创建目录成功")
		} else {
			fmt.Printf("创建目录失败: %v\n", errx)
			return errx
		}
	}
	//计算当天同步的文件个数
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		return err
	}
	number := len(files) + 1
	filename := fmt.Sprintf("%s%s#%d.sql", folderPath, backup.Name, number)
	dumpCmd := fmt.Sprintf("mysqldump -h%s -P%d -u%s -p%s --all-databases > %s", host, port, name, pw, filename)
	command := exec.Command("bash", "-c", dumpCmd)
	fmt.Printf("dumpCmd %v\n", dumpCmd)
	_, err = command.Output() // 执行命令并获取输出
	if err != nil {
		fmt.Printf("Backup failed: %v\n", err)
		return err
	}
	fmt.Printf("DumpMySQL success %v", dumpCmd)

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatormolidocomv1beta1.Databack{}).
		Named("databack").
		Complete(r)
}
