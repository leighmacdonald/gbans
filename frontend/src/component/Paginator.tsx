import * as React from 'react';
import {SyntheticEvent} from 'react';

interface IPageButton {
    current: boolean;
    value: number;
}

interface IPaginatorState {
    current_page: number;
    pages: IPageButton[];
}

interface IPaginatorProps {
    total: number;
    current_page: number;
    onChange: (page: number) => void;
}

export default class Paginator extends React.Component<IPaginatorProps, IPaginatorState> {
    constructor(props: IPaginatorProps) {
        super(props);
        this.onChangePage = this.onChangePage.bind(this);
        this.state = {
            current_page: 1,
            pages: []
        };
    }

    onChangePage(evt: SyntheticEvent, page: number) {
        evt.preventDefault();
        const {total} = this.props;
        if (page > total || page < 0) {
            return;
        }
        this.setState({...this.state, current_page: page});
        this.props.onChange(page);
    }

    onChangeInput(evt: SyntheticEvent) {
        const v = parseInt((evt.target as HTMLInputElement).value, 10);
        if (!isNaN(v)) {
            this.onChangePage(evt, v - 1);
        }
    }

    render_center(pages: IPageButton[]) {
        const b = [];
        for (let i = 0; i < pages.length; i++) {
            if (this.state.current_page == i) {
                b.push(
                    <li key={i} className="current">
                        <span className="show-for-sr">You're on page</span> {i + 1}
                    </li>
                );
            } else {
                b.push(
                    <li key={i}>
                        <a
                            onClick={evt => {
                                this.onChangePage(evt, i);
                            }}
                            href="#"
                            aria-label={`Page ${i + 1}`}
                        >
                            {i + 1}
                        </a>
                    </li>
                );
            }
        }
        return b;
    }

    render_previous() {
        let prevCls = 'pagination-previous';
        if (this.state.current_page === 0) {
            prevCls += ' disabled';
        }
        return (
            <li>
                <a
                    className={prevCls}
                    href="#"
                    aria-label="Previous page"
                    onClick={evt => {
                        this.onChangePage(evt, this.state.current_page - 1);
                    }}
                >
                    Prev
                </a>
            </li>
        );
    }

    render_next() {
        let nextCls = 'pagination-next';
        if (this.state.current_page === 0) {
            nextCls += ' disabled';
        }
        return (
            <li>
                <a
                    className={nextCls}
                    href="#"
                    aria-label="Next page"
                    onClick={evt => {
                        this.onChangePage(evt, this.state.current_page + 1);
                    }}
                >
                    Next
                </a>
            </li>
        );
    }

    render() {
        const pages: IPageButton[] = [];
        for (let i = 0; i < this.props.total; i++) {
            pages.push({
                current: i == this.state.current_page,
                value: i
            });
        }
        return (
            <nav aria-label="Pagination" className={'paginator'}>
                <ul className="pagination text-center">
                    {this.render_previous()}
                    {this.render_center(pages)}
                    {this.render_next()}
                </ul>
            </nav>
        );
    }
}
