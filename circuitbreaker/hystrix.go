package circuitbreaker

import (
	log "github.com/golang/glog"
	"github.com/justintv90/go-micro/endpoint"

	"github.com/afex/hystrix-go/hystrix"
)

func Hystrix(commandName string, runFunc func() error, callbackFunc func(error) error, timeOut, errorPercent, maxConcurrenty, requestVolume, sleep int) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func() error {
			hystrix.ConfigureCommand(commandName, hystrix.CommandConfig{
				Timeout:                timeOut,
				MaxConcurrentRequests:  maxConcurrenty,
				RequestVolumeThreshold: requestVolume,
				SleepWindow:            sleep,
				ErrorPercentThreshold:  errorPercent,
			})
			output := make(chan bool, 1)
			errors := hystrix.Go(commandName, func() error {
				err := runFunc()
				if err == nil {
					output <- true
				}
				return err
			}, callbackFunc)

			select {
			case <-output:
				log.Infoln("Request successful")
				return nil
			case err := <-errors:
				log.Errorln("Request error: ", err)
				return err
			}

		}
	}

}
