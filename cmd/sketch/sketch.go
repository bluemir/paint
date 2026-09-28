package sketch

import (
	"context"
	"fmt"
	"io/fs"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	"github.com/cockroachdb/errors"

	"github.com/bluemir/paint/internal/sketch"
)

// defaultSize 는 --size 없이 새 판을 열 때의 크기다. 터미널 최소 기준이다.
const defaultSize = "80x24"

func Register(cmd *kingpin.CmdClause) {
	var path, size string
	cmd.Arg("path", "그림 파일 (없으면 새로 만든다)").
		Required().
		StringVar(&path)
	// 새 판의 크기다. 있는 파일은 제 크기를 들고 있으므로 같이 주면 거절한다. 크기 바꾸기는 아직
	// 없다.
	cmd.Flag("size", "새 판의 크기, 가로x세로 (기본 "+defaultSize+")").
		StringVar(&size)

	cmd.Action(func(*kingpin.ParseContext) error {
		ctx, stop := signal.NotifyContext(context.Background(),
			syscall.SIGTERM,
			syscall.SIGINT,
		)
		defer stop()

		canvas, err := sketch.Load(path)
		switch {
		case err == nil && size != "":
			return errors.Errorf("%s 는 이미 있다. --size 는 새 판에만 쓴다", path)
		case errors.Is(err, fs.ErrNotExist):
			if size == "" {
				size = defaultSize
			}
			var width, height int
			if _, err := fmt.Sscanf(size, "%dx%d", &width, &height); err != nil || width < 1 || height < 1 {
				return errors.Errorf("--size %q: 가로x세로 꼴이어야 한다 (예: 120x40)", size)
			}
			canvas = sketch.NewCanvas(width, height)
		case err != nil:
			return err
		}

		return sketch.Run(ctx, path, canvas)
	})
}
